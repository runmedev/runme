package codex

import (
	"context"
	"fmt"
	"net"
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
	"github.com/runmedev/runme/v3/pkg/agent/iam"
)

func TestToolBridge_RejectSecondConnection(t *testing.T) {
	bridge := NewToolBridge(nil)
	ts := newTCP4TestServer(t, http.HandlerFunc(bridge.HandleWebsocket))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	conn1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("first websocket dial failed: %v", err)
	}
	defer conn1.Close()

	waitForBridgeConnection(t, bridge)

	_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		t.Fatalf("second websocket dial should fail")
	}
	if resp == nil || resp.StatusCode != 409 {
		t.Fatalf("second websocket status = %v, want 409", respStatus(resp))
	}
}

func TestToolBridge_RejectsConcurrentSecondConnection(t *testing.T) {
	bridge := NewToolBridge(nil)
	ts := newTCP4TestServer(t, http.HandlerFunc(bridge.HandleWebsocket))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")

	type dialResult struct {
		conn *websocket.Conn
		resp *http.Response
		err  error
	}

	results := make(chan dialResult, 2)
	for i := 0; i < 2; i++ {
		go func() {
			conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
			results <- dialResult{conn: conn, resp: resp, err: err}
		}()
	}

	var successes int
	var conflicts int
	for i := 0; i < 2; i++ {
		result := <-results
		if result.err == nil {
			successes++
			defer result.conn.Close()
			continue
		}
		if result.resp != nil && result.resp.StatusCode == http.StatusConflict {
			conflicts++
		}
	}

	if successes != 1 || conflicts != 1 {
		t.Fatalf("got successes=%d conflicts=%d, want 1 success and 1 conflict", successes, conflicts)
	}
}

func TestToolBridge_ForceReplaceSupersedesConnectingConnection(t *testing.T) {
	bridge := NewToolBridge(&iam.AuthContext{})
	ts := newTCP4TestServer(t, http.HandlerFunc(bridge.HandleWebsocket))
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

	writeAuthEnvelope(t, conn2)
	waitForBridgeConnection(t, bridge)

	writeAuthEnvelope(t, conn1)
	_ = conn1.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, _, err := conn1.ReadMessage(); err == nil {
		t.Fatalf("superseded connection should be closed")
	}

	callErrCh := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, err := bridge.Call(ctx, &toolsv1.ToolCallInput{
			Input: &toolsv1.ToolCallInput_ExecuteCode{
				ExecuteCode: &toolsv1.ExecuteCodeRequest{Code: "console.log('ok')"},
			},
		})
		callErrCh <- err
	}()

	_ = conn2.SetReadDeadline(time.Now().Add(2 * time.Second))
	messageType, message, err := conn2.ReadMessage()
	if err != nil {
		t.Fatalf("force_replace connection did not receive bridge call: %v", err)
	}
	if messageType != websocket.TextMessage {
		t.Fatalf("unexpected websocket message type: got %d", messageType)
	}
	req := &codexv1.WebsocketResponse{}
	if err := protojson.Unmarshal(message, req); err != nil {
		t.Fatalf("Unmarshal websocket response failed: %v", err)
	}
	toolReq := req.GetNotebookToolCallRequest()
	if toolReq == nil {
		t.Fatalf("request missing notebook_tool_call_request payload")
	}
	writeBridgeToolResponse(t, conn2, toolReq.GetBridgeCallId())
	if err := <-callErrCh; err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
}

func TestToolBridge_ForceReplaceConnection(t *testing.T) {
	bridge := NewToolBridge(nil)
	ts := newTCP4TestServer(t, http.HandlerFunc(bridge.HandleWebsocket))
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
	bridge := NewToolBridge(nil)
	ts := newTCP4TestServer(t, http.HandlerFunc(bridge.HandleWebsocket))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("websocket dial failed: %v", err)
	}
	defer conn.Close()

	responderErrCh := make(chan error, 1)
	go func() {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			responderErrCh <- fmt.Errorf("ReadMessage request failed: %w", err)
			return
		}
		if messageType != websocket.TextMessage {
			responderErrCh <- fmt.Errorf("unexpected websocket message type: got %d", messageType)
			return
		}
		req := &codexv1.WebsocketResponse{}
		if err := protojson.Unmarshal(message, req); err != nil {
			responderErrCh <- fmt.Errorf("Unmarshal websocket response failed: %w", err)
			return
		}
		toolReq := req.GetNotebookToolCallRequest()
		if toolReq == nil {
			responderErrCh <- fmt.Errorf("request missing notebook_tool_call_request payload")
			return
		}
		parsedInput := toolReq.GetInput()
		if parsedInput.GetExecuteCode() == nil {
			responderErrCh <- fmt.Errorf("request missing execute_code payload")
			return
		}

		resp := &codexv1.WebsocketRequest{
			Payload: &codexv1.WebsocketRequest_NotebookToolCallResponse{
				NotebookToolCallResponse: &codexv1.NotebookToolCallResponse{
					BridgeCallId: toolReq.GetBridgeCallId(),
					Output: &toolsv1.ToolCallOutput{
						CallId: toolReq.GetBridgeCallId(),
						Output: &toolsv1.ToolCallOutput_ExecuteCode{
							ExecuteCode: &toolsv1.ExecuteCodeResponse{Output: "ok\n"},
						},
						Status: toolsv1.ToolCallOutput_STATUS_SUCCESS,
					},
				},
			},
		}
		respJSON, err := protojson.Marshal(resp)
		if err != nil {
			responderErrCh <- fmt.Errorf("Marshal websocket request failed: %w", err)
			return
		}
		if err := conn.WriteMessage(websocket.TextMessage, respJSON); err != nil {
			responderErrCh <- fmt.Errorf("WriteMessage response failed: %w", err)
			return
		}
		responderErrCh <- nil
	}()

	waitForBridgeConnection(t, bridge)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	output, err := bridge.Call(ctx, &toolsv1.ToolCallInput{
		Input: &toolsv1.ToolCallInput_ExecuteCode{
			ExecuteCode: &toolsv1.ExecuteCodeRequest{Code: "console.log('ok')"},
		},
	})
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if output.GetExecuteCode() == nil {
		t.Fatalf("Call output missing execute_code payload")
	}
	if responderErr := <-responderErrCh; responderErr != nil {
		t.Fatal(responderErr)
	}
}

func TestToolBridge_IgnoresUnsupportedPayloads(t *testing.T) {
	bridge := NewToolBridge(nil)
	ts := newTCP4TestServer(t, http.HandlerFunc(bridge.HandleWebsocket))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("websocket dial failed: %v", err)
	}
	defer conn.Close()

	writerErrCh := make(chan error, 1)
	go func() {
		message := &codexv1.WebsocketRequest{}
		data, marshalErr := protojson.Marshal(message)
		if marshalErr != nil {
			writerErrCh <- fmt.Errorf("Marshal websocket request failed: %w", marshalErr)
			return
		}
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			writerErrCh <- fmt.Errorf("WriteMessage failed: %w", err)
			return
		}
		writerErrCh <- nil
	}()

	waitForBridgeConnection(t, bridge)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err = bridge.Call(ctx, &toolsv1.ToolCallInput{
		Input: &toolsv1.ToolCallInput_ExecuteCode{
			ExecuteCode: &toolsv1.ExecuteCodeRequest{Code: "console.log('ok')"},
		},
	})
	if err == nil {
		t.Fatalf("Call should fail when no notebook_tool_call_response is returned")
	}
	if writerErr := <-writerErrCh; writerErr != nil {
		t.Fatal(writerErr)
	}
}

func TestToolBridge_CallRoundTripBinary(t *testing.T) {
	bridge := NewToolBridge(nil)
	ts := newTCP4TestServer(t, http.HandlerFunc(bridge.HandleWebsocket))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("websocket dial failed: %v", err)
	}
	defer conn.Close()

	responderErrCh := make(chan error, 1)
	go func() {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			responderErrCh <- fmt.Errorf("ReadMessage request failed: %w", err)
			return
		}
		if messageType != websocket.TextMessage {
			responderErrCh <- fmt.Errorf("unexpected websocket message type: got %d", messageType)
			return
		}
		req := &codexv1.WebsocketResponse{}
		if err := protojson.Unmarshal(message, req); err != nil {
			responderErrCh <- fmt.Errorf("Unmarshal websocket response failed: %w", err)
			return
		}
		toolReq := req.GetNotebookToolCallRequest()
		if toolReq == nil {
			responderErrCh <- fmt.Errorf("request missing notebook_tool_call_request payload")
			return
		}
		resp := &codexv1.WebsocketRequest{
			Payload: &codexv1.WebsocketRequest_NotebookToolCallResponse{
				NotebookToolCallResponse: &codexv1.NotebookToolCallResponse{
					BridgeCallId: toolReq.GetBridgeCallId(),
					Output: &toolsv1.ToolCallOutput{
						CallId: toolReq.GetBridgeCallId(),
						Output: &toolsv1.ToolCallOutput_ExecuteCode{
							ExecuteCode: &toolsv1.ExecuteCodeResponse{Output: "ok\n"},
						},
						Status: toolsv1.ToolCallOutput_STATUS_SUCCESS,
					},
				},
			},
		}
		data, err := proto.Marshal(resp)
		if err != nil {
			responderErrCh <- fmt.Errorf("Marshal binary response failed: %w", err)
			return
		}
		if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
			responderErrCh <- fmt.Errorf("WriteMessage response failed: %w", err)
			return
		}
		responderErrCh <- nil
	}()

	waitForBridgeConnection(t, bridge)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	output, err := bridge.Call(ctx, &toolsv1.ToolCallInput{
		Input: &toolsv1.ToolCallInput_ExecuteCode{
			ExecuteCode: &toolsv1.ExecuteCodeRequest{Code: "console.log('ok')"},
		},
	})
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if output.GetExecuteCode() == nil {
		t.Fatalf("Call output missing execute_code payload")
	}
	if responderErr := <-responderErrCh; responderErr != nil {
		t.Fatal(responderErr)
	}
}

func TestToolBridge_CallFailsWithoutConnection(t *testing.T) {
	bridge := NewToolBridge(nil)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := bridge.Call(ctx, &toolsv1.ToolCallInput{
		Input: &toolsv1.ToolCallInput_ExecuteCode{
			ExecuteCode: &toolsv1.ExecuteCodeRequest{Code: "console.log('ok')"},
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

func writeAuthEnvelope(t *testing.T, conn *websocket.Conn) {
	t.Helper()
	if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"authorization":"Bearer test"}`)); err != nil {
		t.Fatalf("failed to write auth envelope: %v", err)
	}
}

func writeBridgeToolResponse(t *testing.T, conn *websocket.Conn, bridgeCallID string) {
	t.Helper()
	resp := &codexv1.WebsocketRequest{
		Payload: &codexv1.WebsocketRequest_NotebookToolCallResponse{
			NotebookToolCallResponse: &codexv1.NotebookToolCallResponse{
				BridgeCallId: bridgeCallID,
				Output: &toolsv1.ToolCallOutput{
					CallId: bridgeCallID,
					Output: &toolsv1.ToolCallOutput_ExecuteCode{
						ExecuteCode: &toolsv1.ExecuteCodeResponse{Output: "ok\n"},
					},
					Status: toolsv1.ToolCallOutput_STATUS_SUCCESS,
				},
			},
		},
	}
	data, err := protojson.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal websocket request failed: %v", err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		t.Fatalf("WriteMessage response failed: %v", err)
	}
}

func newTCP4TestServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()

	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		if strings.Contains(err.Error(), "operation not permitted") {
			t.Skipf("skipping websocket listener test in restricted sandbox: %v", err)
		}
		t.Fatalf("failed to start test listener: %v", err)
	}
	server := httptest.NewUnstartedServer(handler)
	server.Listener = listener
	server.Start()
	return server
}

func waitForBridgeConnection(t *testing.T, bridge *ToolBridge) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		bridge.mu.Lock()
		connected := bridge.conn != nil
		bridge.mu.Unlock()
		if connected {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for codex websocket bridge connection")
}
