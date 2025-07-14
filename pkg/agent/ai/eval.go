package ai

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"buf.build/go/protovalidate"
	"connectrpc.com/connect"
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/packages/param"
	"github.com/openai/openai-go/responses"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/yaml.v3"

	agentv1 "github.com/runmedev/runme/v3/api/gen/proto/go/agent/v1"
	"github.com/runmedev/runme/v3/api/gen/proto/go/agent/v1/agentv1connect"

	parserv1 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/parser/v1"
	"github.com/runmedev/runme/v3/pkg/agent/docs"
	"github.com/runmedev/runme/v3/pkg/agent/logs"
	"github.com/runmedev/runme/v3/pkg/agent/version"
)

const (
	llmJudgeInstructions = `
	**Role:** Large-Language-Model (LLM) Judge
	**Purpose:** Evaluate the performance of our AI Site Reliability Engineer (AI SRE).

	### Background
	The AI SRE helps developers deploy and operate their software on the company's internal cloud. It can use several tools—for example:

	- **'bash'** to run shell commands
	- **'filesearch'** to locate internal documents
	- Other task-specific tools as provided

	### Your task
	1. Review the information supplied to you:
	- **Evaluation rubric** listing required behaviours (e.g., did the AI SRE include the '--context' flag when invoking 'kubectl'?).
	- **Conversation or logs** showing what the AI SRE did.
	2. Decide how well the AI SRE met the user's requirements.

	### Output format
	Return a single JSON object with these fields:

		{
		"passed": <boolean>,        // true if the AI SRE satisfies the rubric; otherwise false
		"reasoning": "<string>"     // brief explanation of the pass/fail decision
		}

	- **'passed'** - 'true' when every mandatory criterion is satisfied; otherwise 'false'.
	- **'reasoning'** - concise justification for the result.

	### Rubric
	Below is the rubric for the evaluation:
	`
)

type Asserter interface {
	Assert(ctx context.Context, as *agentv1.Assertion, inputText string, cells map[string]*parserv1.Cell) error
}

func dumpCells(cells map[string]*parserv1.Cell) string {
	var context_builder strings.Builder
	for _, cell := range cells {
		context_builder.WriteString(fmt.Sprintf("Type: %s, Role: %s, Contents: %s\n", cell.Kind, cell.Role, cell.Value))
	}
	return context_builder.String()
}

type shellRequiredFlag struct{}

func (s shellRequiredFlag) Assert(ctx context.Context, as *agentv1.Assertion, inputText string, cells map[string]*parserv1.Cell) error {
	shellFlag := as.GetShellRequiredFlag()
	command := shellFlag.Command
	flags := shellFlag.Flags
	contain_command := false                     // Tracks if the target command is found in any code cell
	as.Result = agentv1.Assertion_RESULT_SKIPPED // Default result is SKIPPED unless the command is found
	for _, cell := range cells {
		if cell.Kind == parserv1.CellKind_CELL_KIND_CODE {
			if strings.Contains(cell.Value, command) { // Check if the code cell contains the target command
				if !contain_command {
					contain_command = true
					as.Result = agentv1.Assertion_RESULT_TRUE // Set to PASSED if the command is present (may be overridden below)
				}
				for _, flag := range flags { // If the command is present, check for all required flags
					if !strings.Contains(cell.Value, flag) {
						as.FailureReason += fmt.Sprintf("Flag %s is missing", flag)
						as.Result = agentv1.Assertion_RESULT_FALSE // Set to FAILED if any required flag is missing
					}
				}
			}
		}
	}
	if as.Result == agentv1.Assertion_RESULT_FALSE {
		as.FailureReason = "Command " + command + " is present, but required flags are missing" + as.FailureReason
	}

	logger, _ := logr.FromContext(ctx)
	logger.Info("shellRequiredFlag", "assertion", as.Name, "result", as.Result)
	return nil
}

type toolInvocation struct{}

func (t toolInvocation) Assert(ctx context.Context, as *agentv1.Assertion, inputText string, cells map[string]*parserv1.Cell) error {
	targetTool := as.GetToolInvocation().GetToolName()
	as.Result = agentv1.Assertion_RESULT_FALSE // Default to false unless the tool is invoked
	for _, cell := range cells {
		// N.B. For now, every tool-call response is treated as code execution in cells.go.
		// TODO: When we add additional tools, handle tool-call responses separately.
		if targetTool == "shell" {
			if cell.Kind == parserv1.CellKind_CELL_KIND_CODE {
				as.Result = agentv1.Assertion_RESULT_TRUE
				break
			}
		} else if targetTool == "file_retrieval" {
			if cell.Kind == parserv1.CellKind_CELL_KIND_DOC_RESULTS {
				as.Result = agentv1.Assertion_RESULT_TRUE
				break
			}
		}
	}
	if as.Result == agentv1.Assertion_RESULT_FALSE {
		as.FailureReason = "Tool " + targetTool + " is not invoked"
	}
	logger, _ := logr.FromContext(ctx)
	logger.Info("toolInvocation", "assertion", as.Name, "result", as.Result)
	return nil
}

type fileRetrieved struct{}

func (f fileRetrieved) Assert(ctx context.Context, as *agentv1.Assertion, inputText string, cells map[string]*parserv1.Cell) error {
	targetFileId := as.GetFileRetrieval().FileId
	as.Result = agentv1.Assertion_RESULT_FALSE // Default to false unless the file is found
	for _, cell := range cells {
		if cell.Kind == parserv1.CellKind_CELL_KIND_DOC_RESULTS {
			for _, file := range cell.DocResults {
				if file.FileId == targetFileId {
					as.Result = agentv1.Assertion_RESULT_TRUE
					break
				}
			}
		}
	}
	if as.Result == agentv1.Assertion_RESULT_FALSE {
		as.FailureReason = "File " + targetFileId + " is not retrieved"
	}
	logger, _ := logr.FromContext(ctx)
	logger.Info("fileRetrieved", "assertion", as.Name, "result", as.Result)
	return nil
}

type llmJudge struct {
	client *openai.Client
}

func NewLlmJudge(client *openai.Client) *llmJudge {
	return &llmJudge{client: client}
}

func (l llmJudge) Assert(ctx context.Context, as *agentv1.Assertion, inputText string, cells map[string]*parserv1.Cell) error {
	logger, _ := logr.FromContext(ctx)
	var context_builder strings.Builder
	for _, cell := range cells {
		markdown := docs.CellToMarkdown(cell, 10000)
		context_builder.WriteString(markdown + "\n")
	}
	logger.Info("llm_judge_debug_input", "input", context_builder.String())
	logger.Info("llm_judge_debug_output", "output", llmJudgeInstructions+as.GetLlmJudge().GetPrompt())
	createResponse := responses.ResponseNewParams{
		Input:        responses.ResponseNewParamsInputUnion{OfString: openai.Opt(context_builder.String())},
		Instructions: openai.Opt(llmJudgeInstructions + as.GetLlmJudge().GetPrompt()),
		Model:        openai.ChatModelO3,
		Text: responses.ResponseTextConfigParam{
			Format: responses.ResponseFormatTextConfigUnionParam{
				OfJSONSchema: &responses.ResponseFormatTextJSONSchemaConfigParam{
					Name: "llm_judge_response",
					Schema: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"passed": map[string]any{
								"type":        "boolean",
								"description": "Whether the assertion passed.",
							},
							"reasoning": map[string]any{
								"type":        "string",
								"description": "Detailed reasoning for the judgement.",
							},
						},
						"required":             []string{"passed", "reasoning"},
						"additionalProperties": false,
					},
					Strict:      param.Opt[bool]{Value: true},
					Description: param.Opt[string]{Value: "Schema for LLM-judge responses"},
				},
			},
		},
	}
	if l.client == nil {
		return errors.New("llmJudge client is not set")
	}
	response, err := l.client.Responses.New(context.Background(), createResponse)
	if err != nil {
		return errors.Wrapf(err, "failed to create response")
	}
	var respMap map[string]any
	err = json.Unmarshal([]byte(response.OutputText()), &respMap)
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal LLM-judge response JSON")
	}
	if passed, ok := respMap["passed"].(bool); ok && passed {
		as.Result = agentv1.Assertion_RESULT_TRUE
	} else {
		as.FailureReason = respMap["reasoning"].(string)
		as.Result = agentv1.Assertion_RESULT_FALSE
	}

	logger.Info("llmJudge", "response", response.OutputText())
	logger.Info("llmJudge", "assertion", as.Name, "result", as.Result)
	return nil
}

type codecellRegex struct{}

func (c codecellRegex) Assert(ctx context.Context, as *agentv1.Assertion, inputText string, cells map[string]*parserv1.Cell) error {
	regexPattern := as.GetCodeblockRegex().Regex
	if regexPattern == "" {
		as.Result = agentv1.Assertion_RESULT_SKIPPED
		return nil
	}
	re, err := regexp.Compile(regexPattern)
	if err != nil {
		as.Result = agentv1.Assertion_RESULT_FALSE
		return errors.Wrapf(err, "invalid regex pattern: %s", regexPattern)
	}
	matched := false
	for _, cell := range cells {
		if cell.Kind == parserv1.CellKind_CELL_KIND_CODE {
			if re.MatchString(cell.Value) {
				matched = true
				break
			}
		}
	}
	if matched {
		as.Result = agentv1.Assertion_RESULT_TRUE
	} else {
		as.FailureReason = "No codecell matches regex: " + regexPattern
		as.Result = agentv1.Assertion_RESULT_FALSE
	}
	logger, _ := logr.FromContext(ctx)
	logger.Info("codecellRegex", "assertion", as.Name, "result", as.Result)
	return nil
}

var registry = map[agentv1.Assertion_Type]Asserter{
	agentv1.Assertion_TYPE_SHELL_REQUIRED_FLAG: shellRequiredFlag{},
	agentv1.Assertion_TYPE_TOOL_INVOKED:        toolInvocation{},
	agentv1.Assertion_TYPE_FILE_RETRIEVED:      fileRetrieved{},
	agentv1.Assertion_TYPE_LLM_JUDGE:           llmJudge{},
	agentv1.Assertion_TYPE_CODEBLOCK_REGEX:     codecellRegex{},
}

func runInference(input string, agentCookie string, inferenceEndpoint string) (map[string]*parserv1.Cell, error) {
	log := zapr.NewLoggerWithOptions(zap.L(), zapr.AllowZapFields(true))

	cells := make(map[string]*parserv1.Cell)

	Cell := parserv1.Cell{
		Kind:  parserv1.CellKind_CELL_KIND_MARKUP,
		Value: "This is a cell",
	}

	log.Info("Cell", logs.ZapProto("cell", &Cell))

	baseURL := inferenceEndpoint
	if baseURL == "" {
		return cells, errors.New("inferenceEndpoint is not set in config")
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		log.Error(err, "Failed to parse URL")
		return cells, errors.Wrapf(err, "Failed to parse URL")
	}

	// Use parserv1.CellKind_CELL_KIND_DOC_RESULTS instead of CELL_KIND_FILE_SEARCH_RESULTS
	// Remove agentv1connect.CellsServiceClient and related client code
	// Update request construction to use Cells, not Cells

	ctx := context.Background()
	genReq := &agentv1.GenerateRequest{
		Cells: []*parserv1.Cell{
			{
				Kind:  parserv1.CellKind_CELL_KIND_MARKUP,
				Role:  parserv1.CellRole_CELL_ROLE_USER,
				Value: input,
			},
		},
	}
	req := connect.NewRequest(genReq)
	cookie := &http.Cookie{
		Name:  "agent-session",
		Value: agentCookie, // supply the real value here, temporary solution
		Path:  "/",         // adjust if needed
	}
	req.Header().Add("Cookie", cookie.String())
	stream, err := agentv1connect.NewMessagesServiceClient(
		&http.Client{
			Transport: &http2.Transport{
				AllowHTTP: true,
				DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
					// Use the standard Dial function to create a plain TCP connection
					return net.Dial(network, u.Host)
				},
			},
		},
		baseURL,
		connect.WithInterceptors(
			connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
				return connect.UnaryFunc(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
					// Add the cookie to the request headers
					req.Header().Add("Cookie", cookie.String())
					return next(ctx, req)
				})
			}),
		),
	).Generate(ctx, req)
	if err != nil {
		return cells, errors.Wrapf(err, "Failed to create generate stream")
	}

	// Receive responses
	for stream.Receive() {
		response := stream.Msg()
		for _, cell := range response.Cells {
			cells[cell.RefId] = cell
		}
	}
	if stream.Err() != nil {
		return cells, errors.Wrapf(stream.Err(), "Error receiving response")
	}
	for _, cell := range cells {
		log.Info(fmt.Sprintf("Received %d cells. Type: %s, Role: %s, Contents: %s", len(cells), cell.Kind, cell.Role, cell.Value))
	}
	return cells, nil
}

// markdownReport holds the data needed to render the evaluation markdown report
type markdownReport struct {
	ExperimentName     string
	DatasetName        string
	NumSamples         int
	NumAssertions      int
	NumPassed          int
	NumFailed          int
	NumSkipped         int
	AssertionTypeStats map[string]struct{ Passed, Failed, Skipped int }
	FailedAssertions   []struct {
		Sample    string
		Assertion string
		Reason    string
		CellsDump string
	}
	Commit    string
	Version   string
	Model     string
	Runner    string
	GoVersion string
	Date      string
}

func (r *markdownReport) Render() string {
	passRate := 100.0
	if r.NumPassed+r.NumFailed > 0 {
		passRate = float64(r.NumPassed) / float64(r.NumPassed+r.NumFailed) * 100
	}
	lines := []string{}
	lines = append(lines, fmt.Sprintf("# AI-SRE Level-1 Evaluation — %s", r.Date))
	lines = append(lines, "")
	lines = append(lines, "| Metric | Value |\n|--------|------:|")
	lines = append(lines, fmt.Sprintf("| Datasets              | `%s` |", r.DatasetName))
	lines = append(lines, fmt.Sprintf("| Samples               | %d |", r.NumSamples))
	lines = append(lines, fmt.Sprintf("| Assertions  | %d |", r.NumAssertions))
	lines = append(lines, fmt.Sprintf("| **Pass rate**         | **%.0f %%** (%d / %d) |", passRate, r.NumPassed, r.NumPassed+r.NumFailed))
	lines = append(lines, "")
	lines = append(lines, "## Pass / fail by assertion type")
	lines = append(lines, "| Assertion | ✅ Passed | ❌ Failed | ⏭️ Skipped | Pass % |")
	lines = append(lines, "|-----------|----------:|---------:|----------:|-------:|")
	// Sort assertion types for stable output
	var types []string
	for typ := range r.AssertionTypeStats {
		types = append(types, typ)
	}
	sort.Strings(types)
	for _, typ := range types {
		stat := r.AssertionTypeStats[typ]
		total := stat.Passed + stat.Failed
		passPct := 0.0
		if total > 0 {
			passPct = float64(stat.Passed) / float64(total) * 100
		}
		lines = append(lines, fmt.Sprintf("| `%s` | %d | %d | %d | %.0f %% |", typ, stat.Passed, stat.Failed, stat.Skipped, passPct))
	}
	lines = append(lines, "")
	if len(r.FailedAssertions) > 0 {
		lines = append(lines, fmt.Sprintf("<details>\n<summary>❌ %d failed assertions (click to expand)</summary>\n", len(r.FailedAssertions)))
		lines = append(lines, "\n| Sample | Assertion | Reason | Cells Dump |\n|--------|-----------|--------|-------------|")
		for _, fail := range r.FailedAssertions {
			escapedDump := strings.ReplaceAll(fail.CellsDump, "\n", "<br/>")
			escapedDump = strings.ReplaceAll(escapedDump, "|", "&#124;")
			escapedReason := strings.ReplaceAll(fail.Reason, "|", "&#124;")
			escapedReason = strings.ReplaceAll(escapedReason, "\n", "<br/>")
			lines = append(lines,
				fmt.Sprintf("| `%s` | `%s` | %s | <details><summary>🔍 View</summary><pre>%s</pre></details> |",
					fail.Sample, fail.Assertion, escapedReason, escapedDump))
		}
		lines = append(lines, "\n</details>\n")
	}
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("_Run metadata: commit `%s`, version `%s`, model `%s`, runner `%s`, %s_", r.Commit, r.Version, r.Model, r.Runner, r.GoVersion))
	return strings.Join(lines, "\n")
}

// EvalFromExperiment runs an experiment based on the Experiment config.
func EvalFromExperiment(exp *agentv1.Experiment, experimentFilePath string, cookie map[string]string, client *openai.Client, log logr.Logger) (map[string]*parserv1.Cell, error) {
	registry[agentv1.Assertion_TYPE_LLM_JUDGE] = NewLlmJudge(client)
	// Resolve dataset path relative to experiment file path if needed
	datasetPath := exp.Spec.GetDatasetPath()
	if !filepath.IsAbs(datasetPath) {
		expDir := filepath.Dir(experimentFilePath)
		datasetPath = filepath.Join(expDir, datasetPath)
	}

	files, err := os.ReadDir(datasetPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read dataset directory %q", datasetPath)
	}

	var samples []*agentv1.EvalSample
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}
		path := filepath.Join(datasetPath, name)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read sample file %q", path)
		}
		var yamlObj interface{}
		if err := yaml.Unmarshal(data, &yamlObj); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal sample yaml file %q", path)
		}
		jsonData, err := json.Marshal(yamlObj)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal sample yaml to json for file %q", path)
		}
		var sample agentv1.EvalSample
		if err := protojson.Unmarshal(jsonData, &sample); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal json to proto for sample file %q", path)
		}
		if err := protovalidate.Validate(&sample); err != nil {
			return nil, errors.Wrapf(err, "failed to validate sample file %q", path)
		}
		samples = append(samples, &sample)
	}

	agentCookie := cookie["agent-session"]
	if agentCookie == "" {
		agentCookie = cookie["cassie-session"]
	}

	inferenceEndpoint := exp.Spec.GetInferenceEndpoint()

	ctx := logr.NewContext(context.Background(), log)

	loc, _ := time.LoadLocation("America/Los_Angeles")
	report := &markdownReport{
		ExperimentName:     exp.Metadata.GetName(),
		DatasetName:        exp.Spec.GetDatasetPath(),
		NumSamples:         len(samples),
		AssertionTypeStats: map[string]struct{ Passed, Failed, Skipped int }{},
		Commit:             version.Commit,
		Version:            version.Version,
		Model:              "gpt-4o-mini", // TODO: fetch dynamically if possible
		Runner:             "linux-amd64", // TODO: fetch dynamically if possible
		GoVersion:          runtime.Version(),
		Date:               time.Now().In(loc).Format("2006-01-02 15:04 MST"),
	}

	totalAssertions := 0
	numPassed := 0
	numFailed := 0
	numSkipped := 0
	failedAssertions := []struct{ Sample, Assertion, Reason, CellsDump string }{}

	for _, sample := range samples {
		cells, err := runInference(sample.InputText, agentCookie, inferenceEndpoint)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to run inference")
		}
		for _, assertion := range sample.Assertions {
			err := registry[assertion.Type].Assert(ctx, assertion, sample.InputText, cells)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to assert %q", assertion.Name)
			}
			totalAssertions++
			typeName := assertion.Type.String()
			stat := report.AssertionTypeStats[typeName]
			switch assertion.Result {
			case agentv1.Assertion_RESULT_TRUE:
				numPassed++
				stat.Passed++
			case agentv1.Assertion_RESULT_FALSE:
				numFailed++
				stat.Failed++
				failedAssertions = append(failedAssertions, struct{ Sample, Assertion, Reason, CellsDump string }{
					Sample:    sample.Metadata.GetName(),
					Assertion: assertion.Name,
					Reason:    assertion.GetFailureReason(),
					CellsDump: dumpCells(cells),
				})
			case agentv1.Assertion_RESULT_SKIPPED:
				numSkipped++
				stat.Skipped++
			}
			report.AssertionTypeStats[typeName] = stat
		}
	}
	report.NumAssertions = totalAssertions
	report.NumPassed = numPassed
	report.NumFailed = numFailed
	report.NumSkipped = numSkipped
	report.FailedAssertions = failedAssertions

	// Write markdown report to outputDir
	outputDir := exp.Spec.GetOutputDir()
	if outputDir == "" {
		outputDir = "."
	}
	if !filepath.IsAbs(outputDir) {
		expDir := filepath.Dir(experimentFilePath)
		outputDir = filepath.Join(expDir, outputDir)
	}
	timestamp := time.Now().In(loc).Format("20060102_150405")
	reportPath := filepath.Join(outputDir, fmt.Sprintf("eval_report_%s.md", timestamp))
	if err := os.WriteFile(reportPath, []byte(report.Render()), 0o644); err != nil {
		return nil, errors.Wrapf(err, "failed to write markdown report to %s", reportPath)
	}

	return nil, nil
}
