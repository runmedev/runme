package ai

import (
	"context"
	"encoding/json"

	"github.com/openai/openai-go/option"

	"connectrpc.com/connect"
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/google/uuid"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/responses"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	parserv1 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/parser/v1"

	"github.com/runmedev/runme/v3/pkg/agent/config"
	"github.com/runmedev/runme/v3/pkg/agent/logs"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"

	agentv1 "github.com/runmedev/runme/v3/api/gen/proto/go/agent/v1"
)

const (
	// DefaultInstructions is the default system prompt to use when generating responses
	DefaultInstructions = `You are an internal Cloud Assistant. Your job is to help developers deploy and operate
their software on their Company's internal cloud. The Cloud consists of Kubernetes clusters, Azure, GitHub, etc...
uses Datadog for monitoring. You have access to CLIs like kubectl, gh, yq, jq, git, az, bazel, curl, wget, etc...
If you need a user to run a command to act or observe the cloud you should respond with the shell tool call.
You also have access to internal documentation which you can use to search for information about
how to use the cloud.

You have access to all the CLIs and tools that Developers use to deploy and operate their software on
the cloud. So you should always try to run commands on a user's behalf and save them the work of invoking
it themselves.

Follow these rules
* Do not rely on outdated documents for determining the status of systems and services.
* Do use the shell tool to run commands to observe the current status of the Cloud
`

	DefaultShellToolDescription = `The shell tool executes CLIs (e.g. kubectl, gh, yq, jq, git, az, bazel, curl, wget, etc...
These CLIs can be used to act and observe on the cloud (Kubernetes, GitHub, Azure, etc...).
The input is a short bash program that can be executed. Additional CLIs can be installed by running the appropriate
commands.

The output of the shell tool is a JSON object with the fields "stderr" and "stdout" containing the output of the
command. If neither of these fields are set then the user hasn't executed the command yet.
`

	ShellToolName = "shell"
)

// Agent implements the AI Service
// https://buf.build/jlewi/foyle/file/main:foyle/v1alpha1/agent.proto#L44
type Agent struct {
	Client               *openai.Client
	instructions         string
	shellToolDescription string
	vectorStoreIDs       []string
	filenameToLink       func(string) string

	// responseCache is a cache to store the mapping from the previous response ID to the cell IDs for function calling
	responseCache *lru.Cache[string, []string]

	// cellsCache is a cache to store the mapping from cellID to cell
	cellsCache *lru.Cache[string, *parserv1.Cell]

	useOAuth bool // Use OAuth for authorization; if true then the token must be provided in the GenerateRequest
}

// AgentOptions are options for creating a new Agent
type AgentOptions struct {
	VectorStores []string
	Client       *openai.Client
	// Instructions are the prompt to use when generating responses
	Instructions string
	// ShellToolDescription is the description of the shell tool.
	ShellToolDescription string

	// FilenameToLink is an optional function that converts a filename to a link to be displayed in the UI.
	FilenameToLink func(string) string

	// UseOAuth indicates whether to use OAuth for authentication
	// If true then the token must be provided in the GenerateRequest
	UseOAuth bool
}

// FromAssistantConfig overrides the AgentOptions based on the values from the AssistantConfig
func (o *AgentOptions) FromAssistantConfig(cfg config.CloudAssistantConfig) error {
	o.VectorStores = cfg.VectorStores

	// TODO(jlewi): We should allow the user to specify the instructions in the config as a path to a file containing
	// the instructions.
	return nil
}

func NewAgent(opts AgentOptions) (*Agent, error) {
	if opts.Client == nil {
		return nil, errors.New("Client is nil")
	}
	log := zapr.NewLogger(zap.L())
	if opts.Instructions == "" {
		opts.Instructions = DefaultInstructions
		log.Info("Using default system prompt")
	}

	if opts.ShellToolDescription == "" {
		opts.ShellToolDescription = DefaultShellToolDescription
		log.Info("Using default shell tool description")
	}

	// Create a cache to store the mapping from the previous response ID to the cell IDs for function calling
	// Should we use an expirable cache?
	responseCache, err := lru.New[string, []string](10000)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create response cache")
	}
	// Create a cache to store the mapping from cellID to cell
	cellsCache, err := lru.New[string, *parserv1.Cell](10000)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create cells cache")
	}

	log.Info("Creating Agent", "options", opts)

	return &Agent{
		Client:               opts.Client,
		instructions:         opts.Instructions,
		shellToolDescription: opts.ShellToolDescription,
		filenameToLink:       opts.FilenameToLink,
		vectorStoreIDs:       opts.VectorStores,
		responseCache:        responseCache,
		cellsCache:           cellsCache,
		useOAuth:             opts.UseOAuth,
	}, nil
}

var shellToolJSONSchema = map[string]any{
	"$schema": "http://json-schema.org/draft-07/schema#",
	"title":   "Shell Function Schema",
	"type":    "object",
	"properties": map[string]interface{}{
		"shell": map[string]interface{}{
			"type":        "string",
			"description": "A short bash program to be executed by bash",
		},
	},
	"required":             []string{"shell"},
	"additionalProperties": false,
}

// Use agentv1.GenerateRequest and agentv1.GenerateResponse, but operate on Cells field.
func (a *Agent) Generate(ctx context.Context, req *connect.Request[agentv1.GenerateRequest], resp *connect.ServerStream[agentv1.GenerateResponse]) error {
	return a.ProcessWithOpenAI(ctx, req.Msg, resp.Send)
}

func (a *Agent) ProcessWithOpenAI(ctx context.Context, req *agentv1.GenerateRequest, sender CellSender) error {
	span := trace.SpanFromContext(ctx)
	log := logs.FromContext(ctx)
	traceId := span.SpanContext().TraceID()
	log = log.WithValues("traceId", traceId)
	ctx = logr.NewContext(ctx, log)
	log.Info("Agent.Generate")

	if (len(req.Cells)) < 1 {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("Cells must be non-empty"))
	}

	tools := make([]responses.ToolUnionParam, 0, 1)

	if len(a.vectorStoreIDs) > 0 {
		fileSearchTool := &responses.FileSearchToolParam{
			MaxNumResults:  openai.Opt(int64(5)),
			VectorStoreIDs: a.vectorStoreIDs,
		}

		tool := responses.ToolUnionParam{
			OfFileSearch: fileSearchTool,
		}
		tools = append(tools, tool)
	}
	shellTool := &responses.FunctionToolParam{
		Name:        ShellToolName,
		Description: openai.Opt(a.shellToolDescription),
		Parameters:  shellToolJSONSchema,
		// N.B. I'm not sure what the point of strict would be since we have a single string argument.
		Strict: openai.Opt(false),
	}

	tool := responses.ToolUnionParam{
		OfFunction: shellTool,
	}
	tools = append(tools, tool)
	// TODO(jlewi): We should add websearch

	// If PreviousResponseId is not set then we need to check that the first cell is user input.
	if req.PreviousResponseId == "" {
		if req.Cells[0].Role != parserv1.CellRole_CELL_ROLE_USER {
			return connect.NewError(connect.CodeInvalidArgument, errors.New("First cell must be user input"))
		}
	}

	toolChoice := responses.ResponseNewParamsToolChoiceUnion{
		OfToolChoiceMode: openai.Opt(responses.ToolChoiceOptionsAuto),
	}

	input := responses.ResponseNewParamsInputUnion{
		// N.B. Input is a list of list. Is that a bug in the SDK
		// ResponseInputParam is a type alias for a list. I find that very confusing.
		OfInputItemList: make([]responses.ResponseInputItemUnionParam, 0, len(req.Cells)),
	}

	if err := fillInToolcalls(ctx, a.responseCache, a.cellsCache, req); err != nil {
		return connect.NewError(connect.CodeInternal, errors.Wrap(err, "Failed to fill in tool calls"))
	}

	for _, c := range req.Cells {
		switch c.Kind {
		case parserv1.CellKind_CELL_KIND_MARKUP:
			input.OfInputItemList = append(input.OfInputItemList, responses.ResponseInputItemUnionParam{
				// N.B. What's the difference between EasyInputMessage and InputItemMessage
				OfMessage: &responses.EasyInputMessageParam{
					Role: responses.EasyInputMessageRoleUser,
					Content: responses.EasyInputMessageContentUnionParam{
						OfString: openai.Opt(c.Value),
					},
				},
			})
		case parserv1.CellKind_CELL_KIND_CODE:
			dict := map[string]string{}

			for _, o := range c.Outputs {
				for _, item := range o.Items {
					// Use item.Type or item.Mime as key, string(item.Data) as value
					key := item.Type
					if key == "" {
						key = item.Mime
					}
					dict[key] += string(item.Data)
				}
			}

			output, err := json.Marshal(dict)
			if err != nil {
				return connect.NewError(connect.CodeInternal, errors.Wrap(err, "Failed to marshal output"))
			}

			shellArgs := &ShellArgs{
				Shell: c.Value,
			}

			shellArgsJSON, err := json.Marshal(shellArgs)
			if err != nil {
				return connect.NewError(connect.CodeInternal, errors.Wrap(err, "Failed to marshal shell args"))
			}

			// The CallID will be blank if it wasn't generated by the model.
			// This can happen if
			// 1. The AI returned code cells in markdown which we parsed out into code cells
			// 2. User manually added the cell
			if c.CallId == "" {
				c.CallId = uuid.NewString()
			}

			// Add the function call to the input
			input.OfInputItemList = append(input.OfInputItemList, responses.ResponseInputItemUnionParam{
				OfFunctionCall: &responses.ResponseFunctionToolCallParam{
					// TODO(jlewi): What if the model didn't tell us to call that function?
					CallID:    c.CallId,
					Name:      ShellToolName,
					Arguments: string(shellArgsJSON),
				},
			})

			input.OfInputItemList = append(input.OfInputItemList, responses.ResponseInputItemUnionParam{
				OfFunctionCallOutput: &responses.ResponseInputItemFunctionCallOutputParam{
					// TODO(jlewi): What if the model didn't tell us to call that function?
					CallID: c.CallId,
					Output: string(output),
				},
			})
		default:
			err := errors.Errorf("Unsupported cell kind %s", c.Kind)
			log.Error(err, "Unsupported cell kind", "cell", c)
			return connect.NewError(connect.CodeInvalidArgument, err)
		}
	}

	createResponse := responses.ResponseNewParams{
		Input:             input,
		Instructions:      openai.Opt(a.instructions),
		Model:             openai.ChatModelGPT4_1,
		Tools:             tools,
		ParallelToolCalls: openai.Bool(true),
		ToolChoice:        toolChoice,
		// We want it to return the file search results
		Include: []responses.ResponseIncludable{responses.ResponseIncludableFileSearchCallResults},
	}

	if req.PreviousResponseId != "" {
		createResponse.PreviousResponseID = openai.Opt(req.PreviousResponseId)
	}

	opts := make([]option.RequestOption, 0, 1)

	if a.useOAuth {
		if req.GetOpenaiAccessToken() == "" {
			log.Info("OpenAI access token is required when using OAuth")
			return connect.NewError(connect.CodeInvalidArgument, errors.New("OpenAI access token is required when using OAuth"))
		}
		opts = append(opts, option.WithHeader("Authorization", "Bearer "+req.GetOpenaiAccessToken()))
	}

	log.Info("ResponseRequest", "request", createResponse)
	eStream := a.Client.Responses.NewStreaming(ctx, createResponse, opts...)
	builder := NewCellsBuilder(a.filenameToLink, a.responseCache, a.cellsCache)

	return builder.HandleEvents(ctx, eStream, sender)
}

// This is necessary because OpenAI returns an error if any of the function calls in the previous response
// are missing output
func fillInToolcalls(ctx context.Context, responseCache *lru.Cache[string, []string], cellsCache *lru.Cache[string, *parserv1.Cell], req *agentv1.GenerateRequest) error {
	if req.PreviousResponseId == "" {
		return nil
	}

	// Check if the previous response ID is in the cache
	prevCalls, ok := responseCache.Get(req.PreviousResponseId)

	// No responses for the previous response ID
	if !ok {
		return nil
	}

	missingPrevCells := make(map[string]bool)
	for _, callID := range prevCalls {
		missingPrevCells[callID] = true
	}

	for _, c := range req.Cells {
		delete(missingPrevCells, c.RefId)
	}

	// If there are any missing function calls then add them
	for callID := range missingPrevCells {
		b, ok := cellsCache.Get(callID)
		if !ok {
			return errors.Errorf("Missing cell for cell ID; %v", callID)
		}
		cellCopy := proto.Clone(b).(*parserv1.Cell)
		req.Cells = append(req.Cells, cellCopy)
	}

	return nil
}
