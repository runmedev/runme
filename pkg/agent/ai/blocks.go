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

	agentv1 "github.com/runmedev/runme/v3/api/gen/proto/go/agent/v1"
	parserv1 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/parser/v1"
	"github.com/runmedev/runme/v3/pkg/agent/docs"
	"github.com/runmedev/runme/v3/pkg/agent/logs"
	"github.com/runmedev/runme/v3/pkg/agent/runme/converters"
)

// CellsBuilder processes the stream of deltas from the responses API and turns them into
// cells to be streamed back to the frontend. This is a stateful operation because responses are deltas
// to be added to previous responses
type CellsBuilder struct {
	filenameToLink func(string) string

	responseCache *lru.Cache[string, []string]
	cellsCache    *lru.Cache[string, *parserv1.Cell]

	responseID string

	// idToCallID is a map from the OpenAI item id to the call_id for function calling.
	// Per the spec https://platform.openai.com/docs/guides/function-calling?api-mode=responses#streaming
	// item ids and call_ids are not the same
	// call_ids are provided on response.output_item.added and response.output_item.done events but not
	// response.function_call_arguments.delta. So we cache them in order to be able to always include the CallID
	// in cells.
	idToCallID map[string]string

	// Map from cell ID to cell
	cells map[string]*parserv1.Cell
	mu    sync.Mutex
}

func NewCellsBuilder(filenameToLink func(string) string, responseCache *lru.Cache[string, []string], cellsCache *lru.Cache[string, *parserv1.Cell]) *CellsBuilder {
	return &CellsBuilder{
		cells:          make(map[string]*parserv1.Cell),
		filenameToLink: filenameToLink,
		responseCache:  responseCache,
		cellsCache:     cellsCache,
		idToCallID:     make(map[string]string),
	}
}

// CellSender is a function that sends a cell to the client
type CellSender func(*agentv1.GenerateResponse) error

// HandleEvents processes a stream of events from the responses API and updates the internal state of the builder
// Function will keep running until the context is cancelled or the stream of events is closed
func (b *CellsBuilder) HandleEvents(ctx context.Context, events *ssestream.Stream[responses.ResponseStreamEventUnion], sender CellSender) error {
	log := logs.FromContext(ctx)
	defer func() {
		resp := &agentv1.GenerateResponse{
			Cells:      make([]*parserv1.Cell, 0, len(b.cells)),
			ResponseId: b.responseID,
		}

		previousIDs := make([]string, 0, len(b.cells))

		for _, cell := range b.cells {
			resp.Cells = append(resp.Cells, cell)

			// Update the cell
			b.cellsCache.Add(cell.RefId, cell)

			// N.B. This ends up including code cells which we parsed out of the markdown and therefore ones which
			// the AI didn't actually generate. Do we want to filter those out?
			if cell.Kind == parserv1.CellKind_CELL_KIND_CODE {
				previousIDs = append(previousIDs, cell.RefId)
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
func (b *CellsBuilder) ProcessEvent(ctx context.Context, e responses.ResponseStreamEventUnion, sender CellSender) error {
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

	resp := &agentv1.GenerateResponse{
		ResponseId: b.responseID,
		Cells:      make([]*parserv1.Cell, 0, 5),
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
		var cell *parserv1.Cell
		ok := false
		cell, ok = b.cells[itemID]
		if !ok {
			cell = &parserv1.Cell{
				RefId: itemID,
				Metadata: map[string]string{
					converters.IdField:      itemID,
					converters.RunmeIdField: itemID,
				},
				Kind:  parserv1.CellKind_CELL_KIND_MARKUP,
				Value: "",
				Role:  parserv1.CellRole_CELL_ROLE_ASSISTANT,
			}
			b.cells[itemID] = cell
		}
		cell.Value += textDelta.Delta
		resp.Cells = append(resp.Cells, cell)

	case responses.ResponseFunctionCallArgumentsDeltaEvent:
		item := e.AsResponseFunctionCallArgumentsDelta()
		itemID := item.ItemID
		if itemID == "" {
			return errors.New("function call arguments delta has no item ID")
		}
		b.mu.Lock()
		defer b.mu.Unlock()
		var cell *parserv1.Cell

		callID, callIDOK := b.idToCallID[itemID]

		if !callIDOK {
			// The call ID should come from the ResponseOutputItemAddedEvent so either there was no
			// ResponseOutputItemAddedEvent or it was missing a call_id.
			return errors.New("function call arguments delta has no call ID")
		}
		ok := false
		cell, ok = b.cells[itemID]
		if !ok {
			// There is no existing cell so we need to initialize a new one.
			cell = &parserv1.Cell{
				RefId: itemID,
				Metadata: map[string]string{
					converters.IdField:      itemID,
					converters.RunmeIdField: itemID,
				},
				Kind:   parserv1.CellKind_CELL_KIND_CODE,
				Value:  "",
				Role:   parserv1.CellRole_CELL_ROLE_ASSISTANT,
				CallId: callID,
			}
			b.cells[itemID] = cell
		}
		// N.B. The delta is the "json string" of the arguments
		// e.g. the deltas will spell out the string {"shell": } character by character
		// So ideally we'd do some kind streaming processing to avoid showing "shell" to the user.
		cell.Value += item.Delta
		resp.Cells = append(resp.Cells, cell)
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
		var cell *parserv1.Cell
		ok := false
		cell, ok = b.cells[itemID]
		if !ok {
			cell = &parserv1.Cell{
				RefId: itemID,
				Metadata: map[string]string{
					converters.IdField:      itemID,
					converters.RunmeIdField: itemID,
				},
				Kind:   parserv1.CellKind_CELL_KIND_CODE,
				Value:  "",
				Role:   parserv1.CellRole_CELL_ROLE_ASSISTANT,
				CallId: callID,
			}
			b.cells[itemID] = cell
		}

		shellArgs := &ShellArgs{}
		if err := json.Unmarshal([]byte(e.Arguments), shellArgs); err != nil {
			log.Error(err, "Failed to unmarshal shell arguments", "delta", e.Arguments)
			cell.Value = e.Arguments
		} else {
			cell.Value = shellArgs.Shell
		}
		resp.Cells = append(resp.Cells, cell)
	case responses.ResponseOutputItemDoneEvent:
		item := e.AsResponseOutputItemDone()
		log.Info(e.Type, "event", e)
		cells, err := b.itemDoneToCell(ctx, item.Item)
		if err != nil {
			return err
		}

		if cells != nil {
			resp.Cells = append(resp.Cells, cells...)
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

	if len(resp.Cells) == 0 {
		log.V(logs.Debug).Info("No cells to send")
		return nil
	}

	if err := sender(resp); err != nil {
		log.Error(err, "Failed to send response")
		return connect.NewError(connect.CodeInternal, errors.Wrapf(err, "Failed to send response to client"))
	}
	return nil
}

func (b *CellsBuilder) itemDoneToCell(ctx context.Context, item responses.ResponseOutputItemUnion) ([]*parserv1.Cell, error) {
	log := logs.FromContext(ctx)
	results := make([]*parserv1.Cell, 0, 5)
	switch item.AsAny().(type) {
	case responses.ResponseOutputMessage:
		// For regular output messages we want to parse out any code cells and turn them into code cells
		// so they get rendered as executable code. This is a bit of a hack to make them executable.
		m := item.AsMessage()
		for _, message := range m.Content {
			if message.Text == "" {
				continue
			}

			parsedCells, err := docs.MarkdownToCells(message.Text)
			if err != nil {
				log.Error(err, "Failed to parse markdown", "text", message.Text)
				continue
			}

			for _, c := range parsedCells {
				if c.Kind == parserv1.CellKind_CELL_KIND_CODE {
					results = append(results, c)
				}
			}
		}
		return results, nil
	case responses.ResponseFileSearchToolCall:
		c, err := b.fileSearchDoneItemToCell(ctx, item.AsFileSearchCall())
		results = append(results, c)
		return results, err
	}
	return results, nil
}

// N.B. It doesn't look like the file search call actually has the results in it. I think its the item done.
func (b *CellsBuilder) fileSearchDoneItemToCell(ctx context.Context, item responses.ResponseFileSearchToolCall) (*parserv1.Cell, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	var cell *parserv1.Cell
	var ok bool
	cell, ok = b.cells[item.ID]
	if !ok {
		cell = &parserv1.Cell{
			RefId: item.ID,
			Metadata: map[string]string{
				converters.IdField:      item.ID,
				converters.RunmeIdField: item.ID,
			},
			Kind:       parserv1.CellKind_CELL_KIND_DOC_RESULTS,
			Role:       parserv1.CellRole_CELL_ROLE_ASSISTANT,
			DocResults: make([]*parserv1.DocResult, 0),
		}
		b.cells[item.ID] = cell
	}

	existing := make(map[string]bool)
	for _, r := range cell.DocResults {
		existing[r.FileId] = true
	}

	for _, r := range item.Results {
		if _, ok := existing[r.FileID]; ok {
			continue
		}

		link := r.Filename
		if b.filenameToLink != nil {
			link = b.filenameToLink(r.Filename)
		}

		cell.DocResults = append(cell.DocResults, &parserv1.DocResult{
			FileId:   r.FileID,
			Score:    r.Score,
			FileName: r.Filename,
			Link:     link,
		})

		existing[r.FileID] = true
	}

	return cell, nil
}
