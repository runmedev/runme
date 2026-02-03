package ai

import (
	"context"
	"encoding/json"

	"github.com/openai/openai-go/option"

	"connectrpc.com/connect"
	"github.com/go-logr/zapr"
	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/responses"
	"github.com/runmedev/runme/v3/api/gen/proto-tools/go/agent/v1/tools/toolsv1mcp"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"

	parserv1 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/parser/v1"

	"github.com/runmedev/runme/v3/pkg/agent/config"
	"github.com/runmedev/runme/v3/pkg/agent/logs"

	"github.com/pkg/errors"

	agentv1 "github.com/runmedev/runme/v3/api/gen/proto/go/agent/v1"
)

const (
	// DefaultInstructions is the default system prompt to use when generating responses
	DefaultInstructions = `You are an internal Cloud Assistant. Your job is to help developers deploy and operate
their software on their Company's internal cloud. The Cloud consists of Kubernetes clusters, Azure, GitHub, etc...
uses Datadog for monitoring. You have access to CLIs like kubectl, gh, yq, jq, git, az, bazel, curl, wget, etc...
If you need a user to run a command to act or observe the cloud you should respond with the code tool call.
You also have access to internal documentation which you can use to search for information about
how to use the cloud.

You have access to all the CLIs and tools that Developers use to deploy and operate their software on
the cloud. So you should always try to run commands on a user's behalf and save them the work of invoking
it themselves.

Follow these rules
* Do not rely on outdated documents for determining the status of systems and services.
* Do use the code tool to run shell commands to observe the current status of the Cloud
`
)

// Agent implements the AI Service
// https://buf.build/jlewi/foyle/file/main:foyle/v1alpha1/agent.proto#L44
type Agent struct {
	Client         *openai.Client
	instructions   string
	vectorStoreIDs []string
	// Extra OpenAI API headers for requests that use OAuth access tokens.
	oauthOpenAIOrganization string
	oauthOpenAIProject      string

	// Tools that should be added based on request context.
	toolsForContext map[agentv1.GenerateRequest_Context][]responses.ToolUnionParam
}

// AgentOptions are options for creating a new Agent
type AgentOptions struct {
	VectorStores []string
	Client       *openai.Client
	// Instructions are the prompt to use when generating responses
	Instructions string

	// Extra OpenAI API headers for requests that use OAuth access tokens.
	OAuthOpenAIOrganization string
	OAuthOpenAIProject      string
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

	toolsForContext := make(map[agentv1.GenerateRequest_Context][]responses.ToolUnionParam)
	nbTools, err := getNotebookTools()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build notebook tools")
	}
	toolsForContext[agentv1.GenerateRequest_CONTEXT_WEBAPP] = nbTools

	runTools, err := getAsynchronousTools()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build asynchronous tools")
	}
	toolsForContext[agentv1.GenerateRequest_CONTEXT_SLACK] = runTools

	log.Info("Creating Agent", "options", opts)

	return &Agent{
		Client:                  opts.Client,
		instructions:            opts.Instructions,
		vectorStoreIDs:          opts.VectorStores,
		oauthOpenAIOrganization: opts.OAuthOpenAIOrganization,
		oauthOpenAIProject:      opts.OAuthOpenAIProject,
		toolsForContext:         toolsForContext,
	}, nil
}

func (a *Agent) Generate(ctx context.Context, req *connect.Request[agentv1.GenerateRequest], resp *connect.ServerStream[agentv1.GenerateResponse]) error {
	// Generate is no longer supported directly; web clients should use the ChatKit path.
	return errors.New("Generate is no longer supported the WebApp should be using chatkit")
}

func (a *Agent) BuildResponseParams(ctx context.Context, req *agentv1.GenerateRequest) (*responses.ResponseNewParams, []option.RequestOption, error) {
	log := logs.FromContext(ctx)
	isOAuthRequest := req.GetOpenaiAccessToken() != ""

	tools := make([]responses.ToolUnionParam, 0, 1)

	if len(a.vectorStoreIDs) > 0 && isOAuthRequest {
		fileSearchTool := &responses.FileSearchToolParam{
			MaxNumResults:  openai.Opt(int64(5)),
			VectorStoreIDs: a.vectorStoreIDs,
		}

		tool := responses.ToolUnionParam{
			OfFileSearch: fileSearchTool,
		}
		tools = append(tools, tool)
	}

	if additional, ok := a.toolsForContext[req.Context]; ok {
		tools = append(tools, additional...)
	}

	if req.GetContainer() != "" {
		log.Info("configuring code interpreter", "containerID", req.GetContainer())
		codeTool := responses.ToolUnionParam{
			OfCodeInterpreter: &responses.ToolCodeInterpreterParam{
				Container: responses.ToolCodeInterpreterContainerUnionParam{
					OfString: openai.Opt(req.GetContainer()),
				},
			},
		}
		tools = append(tools, codeTool)
	}

	toolChoice := responses.ResponseNewParamsToolChoiceUnion{
		OfToolChoiceMode: openai.Opt(responses.ToolChoiceOptionsAuto),
	}

	instructions := a.instructions

	model := openai.ChatModelGPT4oMini
	if req.GetModel() != "" {
		model = openai.ChatModel(req.GetModel())
	}

	createResponse := &responses.ResponseNewParams{
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: make([]responses.ResponseInputItemUnionParam, 0, 10),
		},
		Instructions:      openai.Opt(instructions),
		Model:             model,
		Tools:             tools,
		ParallelToolCalls: openai.Bool(true),
		ToolChoice:        toolChoice,
		Include:           []responses.ResponseIncludable{responses.ResponseIncludableFileSearchCallResults, responses.ResponseIncludableCodeInterpreterCallOutputs},
	}

	// There is a message provide it.
	if req.GetMessage() != "" {
		createResponse.Input.OfInputItemList = append(createResponse.Input.OfInputItemList, responses.ResponseInputItemUnionParam{
			// N.B. What's the difference between EasyInputMessage and InputItemMessage
			OfMessage: &responses.EasyInputMessageParam{
				Role: responses.EasyInputMessageRoleUser,
				Content: responses.EasyInputMessageContentUnionParam{
					OfString: openai.Opt(req.GetMessage()),
				},
			},
		})
	}

	if err := maybeAddListCells(ctx, req, createResponse); err != nil {
		return createResponse, nil, err
	}

	// https://openai.slack.com/archives/C08E32TKF47/p1760471534458089?thread_ts=1760388696.092879&cid=C08E32TKF47
	// Right now chatKit can only handle one tool call at a time. So disable parallel toolCalls.
	if req.GetContext() == agentv1.GenerateRequest_CONTEXT_WEBAPP {
		createResponse.ParallelToolCalls = openai.Opt(false)
	}

	if req.PreviousResponseId != "" {
		createResponse.PreviousResponseID = openai.Opt(req.PreviousResponseId)
	}

	opts := make([]option.RequestOption, 0, 1)
	if isOAuthRequest {
		opts = append(opts, option.WithHeader("Authorization", "Bearer "+req.GetOpenaiAccessToken()))
		if a.oauthOpenAIOrganization == "" {
			return nil, nil, connect.NewError(connect.CodeInvalidArgument, errors.New("OAuth not supported: Server did not configure an API Organization"))
		}
		if a.oauthOpenAIProject == "" {
			return nil, nil, connect.NewError(connect.CodeInvalidArgument, errors.New("OAuth not supported: Server did not configure an API Project"))
		}
		opts = append(opts, option.WithHeader("OpenAI-Organization", a.oauthOpenAIOrganization))
		opts = append(opts, option.WithHeader("OpenAI-Project", a.oauthOpenAIProject))
	}

	return createResponse, opts, nil
}

// getNotebookTools returns a list of tools that allow the AI to work with notebooks.
func getNotebookTools() ([]responses.ToolUnionParam, error) {
	defs := []mcp.Tool{
		toolsv1mcp.NotebookService_GetCellsToolOpenAI,
		toolsv1mcp.NotebookService_ListCellsToolOpenAI,
		toolsv1mcp.NotebookService_UpdateCellsToolOpenAI,
	}

	tools := make([]responses.ToolUnionParam, 0, len(defs))
	for _, t := range defs {
		tool, err := mcpToolToOpenAITool(t)
		if err != nil {
			return nil, err
		}
		tools = append(tools, tool)
	}
	return tools, nil
}

// getAsynchronousTools gets tools used for asynchronous contexts.
func getAsynchronousTools() ([]responses.ToolUnionParam, error) {
	nbTools, err := getNotebookTools()
	if err != nil {
		return nil, err
	}
	tools := make([]responses.ToolUnionParam, 0, len(nbTools)+3)
	tools = append(tools, nbTools...)

	defs := []mcp.Tool{
		toolsv1mcp.NotebookService_ExecuteCellsToolOpenAI,
		toolsv1mcp.NotebookService_TerminateRunToolOpenAI,
		toolsv1mcp.NotebookService_SendSlackMessageToolOpenAI,
	}
	for _, t := range defs {
		tool, err := mcpToolToOpenAITool(t)
		if err != nil {
			return nil, err
		}
		tools = append(tools, tool)
	}
	return tools, nil
}

// mcpToolToOpenAITool converts a generated MCP tool into an OpenAI function tool definition.
func mcpToolToOpenAITool(tool mcp.Tool) (responses.ToolUnionParam, error) {
	result := responses.ToolUnionParam{}
	if len(tool.RawInputSchema) == 0 {
		return result, errors.New("input schema is empty")
	}

	parameters := make(map[string]any)
	if err := json.Unmarshal(tool.RawInputSchema, &parameters); err != nil {
		return result, errors.Wrapf(err, "failed to convert tool: %s", tool.GetName())
	}

	result.OfFunction = &responses.FunctionToolParam{
		Name:        tool.GetName(),
		Description: openai.Opt(tool.Description),
		Parameters:  parameters,
		Strict:      openai.Opt(true),
	}
	return result, nil
}

// maybeAddListCells adds a synthetic list-cells tool call/output so the model gets notebook context.
func maybeAddListCells(_ context.Context, req *agentv1.GenerateRequest, resp *responses.ResponseNewParams) error {
	if len(req.GetCells()) == 0 {
		return nil
	}

	listCellsResult := &agentv1.ListCellsResponse{
		Cells: make([]*parserv1.Cell, 0, len(req.Cells)),
	}
	for _, c := range req.Cells {
		listCellsResult.Cells = append(listCellsResult.Cells, toListCell(c))
	}

	listRequest := &agentv1.ListCellsRequest{}
	listCallID := uuid.NewString()

	listRequestJSON, err := protojson.Marshal(listRequest)
	if err != nil {
		return errors.Wrap(err, "failed to marshal list cells request")
	}

	resp.Input.OfInputItemList = append(resp.Input.OfInputItemList, responses.ResponseInputItemUnionParam{
		OfFunctionCall: &responses.ResponseFunctionToolCallParam{
			CallID:    listCallID,
			Name:      toolsv1mcp.NotebookService_ListCellsToolOpenAI.GetName(),
			Arguments: string(listRequestJSON),
		},
	})

	listResultJSON, err := protojson.Marshal(listCellsResult)
	if err != nil {
		return errors.Wrap(err, "failed to marshal list cells response")
	}

	resp.Input.OfInputItemList = append(resp.Input.OfInputItemList, responses.ResponseInputItemUnionParam{
		OfFunctionCallOutput: &responses.ResponseInputItemFunctionCallOutputParam{
			CallID: listCallID,
			Output: string(listResultJSON),
		},
	})

	return nil
}

// toListCell returns the minimal fields needed for list-cells context.
func toListCell(c *parserv1.Cell) *parserv1.Cell {
	return &parserv1.Cell{
		RefId:    c.GetRefId(),
		Metadata: c.GetMetadata(),
	}
}
