package codex

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	codexv1 "github.com/runmedev/runme/v3/api/gen/proto/go/agent/codex/v1"
	toolsv1 "github.com/runmedev/runme/v3/api/gen/proto/go/agent/tools/v1"
)

func TestToolBridge_RejectSecondConnection(t *testing.T) {
	bridge := NewToolBridge()
	ts := httptest.NewServer(http.HandlerFunc(bridge.HandleWebsocket))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	conn1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("first websocket dial failed: %v", err)
	}
	defer conn1.Close()

	_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		t.Fatalf("second websocket dial should fail")
	}
	if resp == nil || resp.StatusCode != 409 {
		t.Fatalf("second websocket status = %v, want 409", respStatus(resp))
	}
}

func TestToolBridge_ForceReplaceConnection(t *testing.T) {
	bridge := NewToolBridge()
	ts := httptest.NewServer(http.HandlerFunc(bridge.HandleWebsocket))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	conn1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("first websocket dial failed: %v", err)
	}
	defer conn1.Close()

	conn2, _, err := websocket.DefaultDialer.Dial(wsURL+"?force_replace=true", nil)
	if err != nil {
		t.Fatalf("force_replace websocket dial failed: %v", err)
	}
	defer conn2.Close()

	_ = conn1.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, _, err := conn1.ReadMessage(); err == nil {
		t.Fatalf("replaced connection should be closed")
	}
}

func TestToolBridge_CallRoundTrip(t *testing.T) {
	bridge := NewToolBridge()
	ts := httptest.NewServer(http.HandlerFunc(bridge.HandleWebsocket))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("websocket dial failed: %v", err)
	}
	defer conn.Close()

	go func() {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			t.Errorf("ReadMessage request failed: %v", err)
			return
		}
		if messageType != websocket.TextMessage {
			t.Errorf("unexpected websocket message type: got %d", messageType)
			return
		}
		req := &codexv1.WebsocketResponse{}
		if err := protojson.Unmarshal(message, req); err != nil {
			t.Errorf("Unmarshal websocket response failed: %v", err)
			return
		}
		toolReq := req.GetNotebookToolCallRequest()
		if toolReq == nil {
			t.Errorf("request missing notebook_tool_call_request payload")
			return
		}
		parsedInput := toolReq.GetInput()
		if parsedInput.GetListCells() == nil {
			t.Errorf("request missing list_cells payload")
			return
		}

		resp := &codexv1.WebsocketRequest{
			Payload: &codexv1.WebsocketRequest_NotebookToolCallResponse{
				NotebookToolCallResponse: &codexv1.NotebookToolCallResponse{
					BridgeCallId: toolReq.GetBridgeCallId(),
					Output: &toolsv1.ToolCallOutput{
						CallId: toolReq.GetBridgeCallId(),
						Output: &toolsv1.ToolCallOutput_ListCells{
							ListCells: &toolsv1.ListCellsResponse{},
						},
						Status: toolsv1.ToolCallOutput_STATUS_SUCCESS,
					},
				},
			},
		}
		respJSON, err := protojson.Marshal(resp)
		if err != nil {
			t.Errorf("Marshal websocket request failed: %v", err)
			return
		}
		if err := conn.WriteMessage(websocket.TextMessage, respJSON); err != nil {
			t.Errorf("WriteMessage response failed: %v", err)
			return
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	output, err := bridge.Call(ctx, &toolsv1.ToolCallInput{
		Input: &toolsv1.ToolCallInput_ListCells{
			ListCells: &toolsv1.ListCellsRequest{},
		},
	})
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if output.GetListCells() == nil {
		t.Fatalf("Call output missing list_cells payload")
	}
}

func TestToolBridge_IgnoresUnsupportedPayloads(t *testing.T) {
	bridge := NewToolBridge()
	ts := httptest.NewServer(http.HandlerFunc(bridge.HandleWebsocket))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("websocket dial failed: %v", err)
	}
	defer conn.Close()

	go func() {
		message := &codexv1.WebsocketRequest{}
		data, marshalErr := protojson.Marshal(message)
		if marshalErr != nil {
			t.Errorf("Marshal websocket request failed: %v", marshalErr)
			return
		}
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			t.Errorf("WriteMessage failed: %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err = bridge.Call(ctx, &toolsv1.ToolCallInput{
		Input: &toolsv1.ToolCallInput_ListCells{
			ListCells: &toolsv1.ListCellsRequest{},
		},
	})
	if err == nil {
		t.Fatalf("Call should fail when no notebook_tool_call_response is returned")
	}
}

func TestToolBridge_CallRoundTripBinary(t *testing.T) {
	bridge := NewToolBridge()
	ts := httptest.NewServer(http.HandlerFunc(bridge.HandleWebsocket))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("websocket dial failed: %v", err)
	}
	defer conn.Close()

	go func() {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			t.Errorf("ReadMessage request failed: %v", err)
			return
		}
		if messageType != websocket.TextMessage {
			t.Errorf("unexpected websocket message type: got %d", messageType)
			return
		}
		req := &codexv1.WebsocketResponse{}
		if err := protojson.Unmarshal(message, req); err != nil {
			t.Errorf("Unmarshal websocket response failed: %v", err)
			return
		}
		toolReq := req.GetNotebookToolCallRequest()
		if toolReq == nil {
			t.Errorf("request missing notebook_tool_call_request payload")
			return
		}
		resp := &codexv1.WebsocketRequest{
			Payload: &codexv1.WebsocketRequest_NotebookToolCallResponse{
				NotebookToolCallResponse: &codexv1.NotebookToolCallResponse{
					BridgeCallId: toolReq.GetBridgeCallId(),
					Output: &toolsv1.ToolCallOutput{
						CallId: toolReq.GetBridgeCallId(),
						Output: &toolsv1.ToolCallOutput_ListCells{
							ListCells: &toolsv1.ListCellsResponse{},
						},
						Status: toolsv1.ToolCallOutput_STATUS_SUCCESS,
					},
				},
			},
		}
		data, err := proto.Marshal(resp)
		if err != nil {
			t.Errorf("Marshal binary response failed: %v", err)
			return
		}
		if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
			t.Errorf("WriteMessage response failed: %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	output, err := bridge.Call(ctx, &toolsv1.ToolCallInput{
		Input: &toolsv1.ToolCallInput_ListCells{
			ListCells: &toolsv1.ListCellsRequest{},
		},
	})
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if output.GetListCells() == nil {
		t.Fatalf("Call output missing list_cells payload")
	}
}

func TestToolBridge_CallFailsWithoutConnection(t *testing.T) {
	bridge := NewToolBridge()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := bridge.Call(ctx, &toolsv1.ToolCallInput{
		Input: &toolsv1.ToolCallInput_ListCells{
			ListCells: &toolsv1.ListCellsRequest{},
		},
	})
	if err == nil {
		t.Fatalf("Call should fail when bridge is disconnected")
	}
}

func respStatus(resp *http.Response) int {
	if resp == nil {
		return 0
	}
	return resp.StatusCode
}
