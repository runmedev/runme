package ai

import (
	"context"
	"encoding/json"
	"sync"

	"connectrpc.com/connect"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/openai/openai-go/packages/ssestream"
	"github.com/openai/openai-go/responses"
	"github.com/pkg/errors"

	"github.com/runmedev/runme/v3/api/gen/proto/go/agent"
	"github.com/runmedev/runme/v3/pkg/agent/docs"
	"github.com/runmedev/runme/v3/pkg/agent/logs"
)

// BlocksBuilder processes the stream of deltas from the responses API and turns them into
// blocks to be streamed back to the frontend. This is a stateful operation because responses are deltas
// to be added to previous responses
type BlocksBuilder struct {
	filenameToLink func(string) string

	responseCache *lru.Cache[string, []string]
	blocksCache   *lru.Cache[string, *agent.Block]

	responseID string

	// idToCallID is a map from the OpenAI item id to the call_id for function calling.
	// Per the spec https://platform.openai.com/docs/guides/function-calling?api-mode=responses#streaming
	// item ids and call_ids are not the same
	// call_ids are provided on response.output_item.added and response.output_item.done events but not
	// response.function_call_arguments.delta. So we cache them in order to be able to always include the CallID
	// in blocks.
	idToCallID map[string]string

	// Map from block ID to block
	blocks map[string]*agent.Block
	mu     sync.Mutex
}

func NewBlocksBuilder(filenameToLink func(string) string, responseCache *lru.Cache[string, []string], blocksCache *lru.Cache[string, *agent.Block]) *BlocksBuilder {
	return &BlocksBuilder{
		blocks:         make(map[string]*agent.Block),
		filenameToLink: filenameToLink,
		responseCache:  responseCache,
		blocksCache:    blocksCache,
		idToCallID:     make(map[string]string),
	}
}

// BlockSender is a function that sends a block to the client
type BlockSender func(*agent.GenerateResponse) error

// HandleEvents processes a stream of events from the responses API and updates the internal state of the builder
// Function will keep running until the context is cancelled or the stream of events is closed
func (b *BlocksBuilder) HandleEvents(ctx context.Context, events *ssestream.Stream[responses.ResponseStreamEventUnion], sender BlockSender) error {
	log := logs.FromContext(ctx)
	defer func() {
		resp := &agent.GenerateResponse{
			Blocks:     make([]*agent.Block, 0, len(b.blocks)),
			ResponseId: b.responseID,
		}

		previousIDs := make([]string, 0, len(b.blocks))

		for _, block := range b.blocks {
			resp.Blocks = append(resp.Blocks, block)

			// Update the block
			b.blocksCache.Add(block.Id, block)

			// N.B. This ends up including code blocks which we parsed out of the markdown and therefore ones which
			// the AI didn't actually generate. Do we want to filter those out?
			if block.Kind == agent.BlockKind_CODE {
				previousIDs = append(previousIDs, block.Id)
			}
		}

		b.responseCache.Add(resp.ResponseId, previousIDs)
		// Log the final response.
		log.Info("GenerateResponse", logs.ZapProto("response", resp))
	}()

	for events.Next() {
		select {
		// Terminate because the request got cancelled
		case <-ctx.Done():
			log.Info("Context cancelled; stopping streaming request", "err", ctx.Err())
			if errors.Is(ctx.Err(), context.Canceled) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
				// N.B. If the context was cancelled then we should return a DeadlineExceeded error to indicate we hit
				// a timeout on the server.
				// My assumption is if the client terminates the connection there is a different error.
				return connect.NewError(connect.CodeDeadlineExceeded, errors.Wrapf(ctx.Err(), "The request context was cancelled. This usually happens because the read or write timeout of the HTTP server was reched."))
			}
			// Cancel functions will be called when this function returns
			return ctx.Err()
		default:
			// Process the event
			event := events.Current()
			if err := b.ProcessEvent(ctx, event, sender); err != nil {
				log.Error(err, "Error processing event")
				return connect.NewError(connect.CodeInternal, errors.Wrapf(err, "Error processing event"))
			}
		}
	}

	if err := events.Err(); err != nil {
		log.Error(err, "Error processing events")
		return connect.NewError(connect.CodeInternal, errors.Wrapf(err, "Error processing events"))
	}
	return nil
}

// ProcessEvent processes a response stream event and updates the internal state of the builder
func (b *BlocksBuilder) ProcessEvent(ctx context.Context, e responses.ResponseStreamEventUnion, sender BlockSender) error {
	log := logs.FromContext(ctx)
	log.V(logs.Debug).Info("Processing event", "event", e)

	// Per the APISpec the ResponseID is not set on all messages.
	// https://platform.openai.com/docs/api-reference/responses-streaming/response
	// So we store it and then attach it to all responses that we stream back.
	if e.Response.ID != "" {
		if b.responseID == "" {
			b.responseID = e.Response.ID
		} else {
			if b.responseID != e.Response.ID {
				log.Error(errors.New("response ID changed mid-stream"), "old", b.responseID, "new", e.Response.ID)
			}
		}
	}

	resp := &agent.GenerateResponse{
		ResponseId: b.responseID,
		Blocks:     make([]*agent.Block, 0, 5),
	}

	switch e.AsAny().(type) {
	case responses.ResponseOutputItemAddedEvent:
		item := e.AsResponseOutputItemAdded()
		if item.Item.CallID != "" {
			b.idToCallID[item.Item.ID] = item.Item.CallID
		}
	case responses.ResponseContentPartDoneEvent:
		log.Info(e.Type, "event", e)
	case responses.ResponseTextDeltaEvent:
		textDelta := e.AsResponseOutputTextDelta()
		itemID := textDelta.ItemID
		if itemID == "" {
			return errors.New("text delta has no item ID")
		}

		b.mu.Lock()
		defer b.mu.Unlock()
		var block *agent.Block
		ok := false
		block, ok = b.blocks[itemID]
		if !ok {
			block = &agent.Block{
				Id:       itemID,
				Kind:     agent.BlockKind_MARKUP,
				Contents: "",
				Role:     agent.BlockRole_BLOCK_ROLE_ASSISTANT,
			}
			b.blocks[itemID] = block
		}
		block.Contents += textDelta.Delta
		resp.Blocks = append(resp.Blocks, block)

	case responses.ResponseFunctionCallArgumentsDeltaEvent:
		item := e.AsResponseFunctionCallArgumentsDelta()
		itemID := item.ItemID
		if itemID == "" {
			return errors.New("function call arguments delta has no item ID")
		}
		b.mu.Lock()
		defer b.mu.Unlock()
		var block *agent.Block

		callID, callIDOK := b.idToCallID[itemID]

		if !callIDOK {
			// The call ID should come from the ResponseOutputItemAddedEvent so either there was no
			// ResponseOutputItemAddedEvent or it was missing a call_id.
			return errors.New("function call arguments delta has no call ID")
		}
		ok := false
		block, ok = b.blocks[itemID]
		if !ok {
			// There is no existing block so we need to initialize a new one.
			block = &agent.Block{
				Id:       itemID,
				Kind:     agent.BlockKind_CODE,
				Contents: "",
				Role:     agent.BlockRole_BLOCK_ROLE_ASSISTANT,
				CallId:   callID,
			}
			b.blocks[itemID] = block
		}
		// N.B. The delta is the "json string" of the arguments
		// e.g. the deltas will spell out the string {"shell": } character by character
		// So ideally we'd do some kind streaming processing to avoid showing "shell" to the user.
		block.Contents += item.Delta
		resp.Blocks = append(resp.Blocks, block)
	case responses.ResponseFunctionCallArgumentsDoneEvent:
		log.Info(e.Type, "event", e)
		item := e.AsResponseFunctionCallArgumentsDone()
		itemID := item.ItemID
		if itemID == "" {
			return errors.New("function call arguments delta has no item ID")
		}
		callID, callIDok := b.idToCallID[itemID]

		if !callIDok {
			// The call ID should come from the ResponseOutputItemAddedEvent so either there was no
			// ResponseOutputItemAddedEvent or it was missing a call_id.
			return errors.New("function call arguments delta has no call ID")
		}
		b.mu.Lock()
		defer b.mu.Unlock()
		var block *agent.Block
		ok := false
		block, ok = b.blocks[itemID]
		if !ok {
			block = &agent.Block{
				Id:       itemID,
				Kind:     agent.BlockKind_CODE,
				Contents: "",
				Role:     agent.BlockRole_BLOCK_ROLE_ASSISTANT,
				CallId:   callID,
			}
			b.blocks[itemID] = block
		}

		shellArgs := &ShellArgs{}
		if err := json.Unmarshal([]byte(e.Arguments), shellArgs); err != nil {
			log.Error(err, "Failed to unmarshal shell arguments", "delta", e.Arguments)
			block.Contents = e.Arguments
		} else {
			block.Contents = shellArgs.Shell
		}
		resp.Blocks = append(resp.Blocks, block)
	case responses.ResponseOutputItemDoneEvent:
		item := e.AsResponseOutputItemDone()
		log.Info(e.Type, "event", e)
		blocks, err := b.itemDoneToBlock(ctx, item.Item)
		if err != nil {
			return err
		}

		if blocks != nil {
			resp.Blocks = append(resp.Blocks, blocks...)
		}

	case responses.ResponseTextDoneEvent:
		log.Info(e.Type, "event", e)
	case responses.ResponseCompletedEvent:
		// Log the final response
		log.Info(e.Type, "event", e)
	default:
		log.Info("Ignoring event", "event", e)
		log.V(logs.Debug).Info("Ignoring event", "event", e)
	}

	if len(resp.Blocks) == 0 {
		log.V(logs.Debug).Info("No blocks to send")
		return nil
	}

	if err := sender(resp); err != nil {
		log.Error(err, "Failed to send response")
		return connect.NewError(connect.CodeInternal, errors.Wrapf(err, "Failed to send response to client"))
	}
	return nil
}

func (b *BlocksBuilder) itemDoneToBlock(ctx context.Context, item responses.ResponseOutputItemUnion) ([]*agent.Block, error) {
	log := logs.FromContext(ctx)
	results := make([]*agent.Block, 0, 5)
	switch item.AsAny().(type) {
	case responses.ResponseOutputMessage:
		// For regular output messages we want to parse out any code blocks and turn them into code blocks
		// so they get rendered as executable code. This is a bit of a hack to make them executable.
		m := item.AsMessage()
		for _, message := range m.Content {
			if message.Text == "" {
				continue
			}

			parsedBlocks, err := docs.MarkdownToBlocks(message.Text)
			if err != nil {
				log.Error(err, "Failed to parse markdown", "text", message.Text)
				continue
			}

			for _, b := range parsedBlocks {
				if b.Kind == agent.BlockKind_CODE {
					results = append(results, b)
				}
			}
		}
		return results, nil
	case responses.ResponseFileSearchToolCall:
		b, err := b.fileSearchDoneItemToBlock(ctx, item.AsFileSearchCall())
		results = append(results, b)
		return results, err
	}
	return results, nil
}

// N.B. It doesn't look like the file search call actually has the results in it. I think its the item done.
func (b *BlocksBuilder) fileSearchDoneItemToBlock(ctx context.Context, item responses.ResponseFileSearchToolCall) (*agent.Block, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	var block *agent.Block
	var ok bool
	block, ok = b.blocks[item.ID]
	if !ok {
		block = &agent.Block{
			Id:                item.ID,
			Kind:              agent.BlockKind_FILE_SEARCH_RESULTS,
			Role:              agent.BlockRole_BLOCK_ROLE_ASSISTANT,
			FileSearchResults: make([]*agent.FileSearchResult, 0),
		}
		b.blocks[item.ID] = block
	}

	existing := make(map[string]bool)
	for _, r := range block.FileSearchResults {
		existing[r.FileID] = true
	}

	for _, r := range item.Results {
		if _, ok := existing[r.FileID]; ok {
			continue
		}

		link := r.Filename
		if b.filenameToLink != nil {
			link = b.filenameToLink(r.Filename)
		}

		block.FileSearchResults = append(block.FileSearchResults, &agent.FileSearchResult{
			FileID:   r.FileID,
			Score:    r.Score,
			FileName: r.Filename,
			Link:     link,
		})

		existing[r.FileID] = true
	}

	return block, nil
}
