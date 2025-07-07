package e2etests

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/openai/openai-go"

	agentv1 "github.com/runmedev/runme/v3/api/gen/proto/go/agent/v1"
	"github.com/runmedev/runme/v3/pkg/agent/ai"
	"github.com/runmedev/runme/v3/pkg/agent/application"
	"github.com/runmedev/runme/v3/pkg/agent/config"

	"github.com/go-logr/zapr"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
)

func Test_Agent(t *testing.T) {
	// This test verifies that function calling works.
	// We run it in GHA or locally if RUN_MANUAL_TESTS is set
	isGHA := os.Getenv("GITHUB_ACTIONS") == "true"
	if !isGHA {
		SkipIfMissing(t, "RUN_MANUAL_TESTS")
	}
	ghaOwner := os.Getenv("GITHUB_REPOSITORY_OWNER")
	if ghaOwner == "runmedev" {
		t.Skip("Skipping agent test in runmedev repositories")
	}

	// todo(sebastian): we might want to use a different app name for agents/evals
	app := application.NewApp("runme-agent-test")
	if err := app.LoadConfig(nil); err != nil {
		t.Fatal(err)
	}
	if err := app.SetupLogging(); err != nil {
		t.Fatal(err)
	}
	if err := app.SetupOTEL(); err != nil {
		t.Fatal(err)
	}
	cfg := app.GetConfig()

	var client *openai.Client
	var err error

	agentOptions := &ai.AgentOptions{}
	var agentConfg config.CloudAssistantConfig
	if !isGHA {
		// When running locally create the OpenAI client using the config
		client, err = ai.NewClient(*cfg.OpenAI)
		if err != nil {
			t.Fatalf("Failed to create client from application configuration; %v", err)
		}
		agentConfg = *cfg.CloudAssistant
	} else {
		// In GHA we get the API key from the environment variable
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			t.Fatal("OPENAI_API_KEY environment variable is not set")
		}
		client, err = ai.NewClientWithKey(apiKey)
		if err != nil {
			t.Fatalf("Failed to create client from environment variable OPENAI_API_KEY; %v", err)
		}
		agentConfg.VectorStores = []string{"vs_67a829aae998819189b2ba29cef645f6"}
	}

	if err := agentOptions.FromAssistantConfig(agentConfg); err != nil {
		t.Fatalf("Failed to create agent options; %v", err)
	}
	agentOptions.Client = client

	agent, err := ai.NewAgent(*agentOptions)
	if err != nil {
		t.Fatalf("Failed to create agent: %+v", err)
	}
	req := &agentv1.GenerateRequest{
		Blocks: []*agentv1.Block{
			{
				Id:       "1",
				Contents: "Use kubectl to tell me the current status of the rube-dev deployment in the a0s context? Do not rely on outdated documents.",
				Role:     agentv1.BlockRole_BLOCK_ROLE_USER,
				Kind:     agentv1.BlockKind_BLOCK_KIND_MARKUP,
			},
		},
	}

	resp := &ServerResponseStream{
		Events: make([]*agentv1.GenerateResponse, 0, 10),
		Blocks: make(map[string]*agentv1.Block),
	}

	if err := agent.ProcessWithOpenAI(context.Background(), req, resp.Send); err != nil {
		t.Fatalf("Error processing request: %+v", err)
	}

	exCommand, err := regexp.Compile(`kubectl.*get.*deployment.*`)
	if err != nil {
		t.Fatalf("Error compiling regex: %v", err)
	}

	// Check if there is a code execution block
	codeBlocks := make([]*agentv1.Block, 0, len(resp.Blocks))
	for _, b := range resp.Blocks {
		if b.Kind == agentv1.BlockKind_BLOCK_KIND_CODE {
			t.Logf("Found code block with ID: %s", b.Id)
			// Optionally, you can check the contents of the block
			t.Logf("Code block contents: %s", b.Contents)
			matched := exCommand.Match([]byte(b.Contents))

			if !matched {
				t.Errorf("Code block does not match expected pattern; got\n%v", b.Contents)
				continue
			}

			// Rewrite the command so we know its a command that's safe to execute automatically
			b.Contents = `kubectl --context=a0s -n rube get deployment rube-dev -o yaml`
			codeBlocks = append(codeBlocks, b)
		}
	}

	if len(codeBlocks) == 0 {
		t.Fatalf("No code blocks found in response")
	}

	for _, b := range codeBlocks {
		if b.CallId == "" {
			t.Fatalf("Code block with ID %s does not have a CallId", b.Id)
		}
	}

	// Now lets execute a command and provide it to the AI to see how it responds.
	if err := executeBlock(codeBlocks[0], true); err != nil {
		t.Fatalf("Failed to execute command: %+v", err)
	}

	previousResponseID := resp.Events[0].ResponseId
	if previousResponseID == "" {
		t.Fatalf("Previous response ID is empty")
	}
	// Now we need to send the output back to the AI
	codeReq := &agentv1.GenerateRequest{
		PreviousResponseId: previousResponseID,
		Blocks: []*agentv1.Block{
			codeBlocks[0],
		},
	}

	codeResp := &ServerResponseStream{
		Events: make([]*agentv1.GenerateResponse, 0, 10),
		Blocks: make(map[string]*agentv1.Block),
	}

	if err := agent.ProcessWithOpenAI(context.Background(), codeReq, codeResp.Send); err != nil {
		t.Fatalf("Error processing request: %+v", err)
	}

	for _, b := range codeResp.Blocks {
		o := protojson.MarshalOptions{
			Multiline: true,
			Indent:    "  ",
		}
		j, err := o.Marshal(b)
		if err != nil {
			t.Fatalf("Failed to marshal block: %+v", err)
		}
		t.Logf("Block:\n%+v", string(j))
	}
}

type ServerResponseStream struct {
	Events []*agentv1.GenerateResponse
	Blocks map[string]*agentv1.Block
	mu     sync.Mutex
}

func (s *ServerResponseStream) Send(e *agentv1.GenerateResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Events = append(s.Events, e)

	for _, b := range e.Blocks {
		s.Blocks[b.Id] = b
	}
	return nil
}

// Run the given command and return the output
// This can either run the command for real or return canned outputs
func executeBlock(b *agentv1.Block, useCached bool) error {
	log := zapr.NewLogger(zap.L())
	args := strings.Split(b.Contents, " ")

	stdOutStr := cannedStdout
	stdErrStr := ""

	if !useCached {
		cmd := exec.Command(args[0], args[1:]...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err != nil {
			log.Error(err, "Failed to run command", "cmd", b.Contents, "stdout", stdout.String(), "stderr", stderr.String())
			return errors.Wrapf(err, "Failed to run %s", b.Contents)
		}

		log.Info("Command completed successfully", "stdout", stdout.String(), "stderr", stderr.String())

		if stdout.Len() == 0 && stderr.Len() == 0 {
			return errors.New("No output from command")
		}

		stdOutStr = stdout.String()
		stdErrStr = stderr.String()

	}
	b.Outputs = []*agentv1.BlockOutput{
		{
			Kind: agentv1.BlockOutputKind_BLOCK_OUTPUT_KIND_STDOUT,
			Items: []*agentv1.BlockOutputItem{
				{
					TextData: stdOutStr,
				},
			},
		},
		{
			Kind: agentv1.BlockOutputKind_BLOCK_OUTPUT_KIND_STDERR,
			Items: []*agentv1.BlockOutputItem{
				{
					TextData: stdErrStr,
				},
			},
		},
	}

	return nil
}

const (
	cannedStdout = `apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    deployment.kubernetes.io/revision: "9"
  creationTimestamp: "2024-12-10T19:36:25Z"
  generation: 20
  labels:
    app: rube
    env: dev
  name: rube-dev
  namespace: rube
  resourceVersion: "629424835"
  uid: 73e83e63-e160-4e86-ab5f-e5ba99d25266
spec:
  progressDeadlineSeconds: 600
  replicas: 1
  revisionHistoryLimit: 10
  selector:
    matchLabels:
      app: rube
      env: dev
  strategy:
    rollingUpdate:
      maxSurge: 25%
      maxUnavailable: 25%
    type: RollingUpdate
  template:
    metadata:
      annotations:
        kubectl.kubernetes.io/restartedAt: "2024-12-10T14:30:53-08:00"
      creationTimestamp: null
      labels:
        app: rube
        env: dev
    spec:
      containers:
      - args:
        - --openai-apikey=/etc/secrets/openai-key/openai.api.key
        image: ghcr.io/jlewi/rube/rube@sha256:a60e69f9fbf9995a78db8bb457b3b4fba6ef400825188ea67e213e0c966ceff6
        imagePullPolicy: IfNotPresent
        livenessProbe:
          failureThreshold: 3
          httpGet:
            path: /healthz
            port: 8080
            scheme: HTTP
          initialDelaySeconds: 15
          periodSeconds: 10
          successThreshold: 1
          timeoutSeconds: 1
        name: rube
        ports:
        - containerPort: 8080
          protocol: TCP
        readinessProbe:
          failureThreshold: 3
          httpGet:
            path: /healthz
            port: 8080
            scheme: HTTP
          initialDelaySeconds: 10
          periodSeconds: 5
          successThreshold: 1
          timeoutSeconds: 1
        resources:
          limits:
            cpu: 250m
            memory: 256Mi
          requests:
            cpu: 250m
            memory: 256Mi
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
        volumeMounts:
        - mountPath: /etc/secrets/openai-key
          name: openai-key
          readOnly: true
      dnsPolicy: ClusterFirst
      restartPolicy: Always
      schedulerName: default-scheduler
      securityContext: {}
      serviceAccount: rube
      serviceAccountName: rube
      terminationGracePeriodSeconds: 30
      volumes:
      - name: openai-key
        secret:
          defaultMode: 420
          secretName: openai-key
status:
  conditions:
  - lastTransitionTime: "2025-04-09T23:31:59Z"
    lastUpdateTime: "2025-04-09T23:32:14Z"
    message: ReplicaSet "rube-dev-8569595945" has successfully progressed.
    reason: NewReplicaSetAvailable
    status: "True"
    type: Progressing
  - lastTransitionTime: "2025-05-01T23:41:09Z"
    lastUpdateTime: "2025-05-01T23:41:09Z"
    message: Deployment does not have minimum availability.
    reason: MinimumReplicasUnavailable
    status: "False"
    type: Available
  observedGeneration: 20
  replicas: 1
  unavailableReplicas: 1
  updatedReplicas: 1
`
)
