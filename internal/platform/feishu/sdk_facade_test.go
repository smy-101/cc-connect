package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	wsconn "github.com/gorilla/websocket"
	lark "github.com/larksuite/oapi-sdk-go/v3"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"
)

func TestRealSDKFacadeHelpers(t *testing.T) {
	created := newRealSDKFacade("app-id", "app-secret")
	realFacade, ok := created.(*realSDKFacade)
	if !ok {
		t.Fatalf("newRealSDKFacade() returned %T", created)
	}
	if realFacade.baseURL != lark.FeishuBaseUrl {
		t.Fatalf("baseURL = %q, want %q", realFacade.baseURL, lark.FeishuBaseUrl)
	}

	realFacade.applyConfig(&larkws.ClientConfig{
		ReconnectNonce:    1,
		ReconnectInterval: 3,
		PingInterval:      4,
	})
	if realFacade.reconnectInterval != 3*time.Second {
		t.Fatalf("reconnectInterval = %v", realFacade.reconnectInterval)
	}
	if realFacade.pingInterval != 4*time.Second {
		t.Fatalf("pingInterval = %v", realFacade.pingInterval)
	}
	if realFacade.nextReconnectDelay() < realFacade.reconnectInterval {
		t.Fatal("nextReconnectDelay() should not be less than reconnectInterval")
	}

	if !isRetryableConnectError(fmt.Errorf("temporary network issue")) {
		t.Fatal("expected network error to be retryable")
	}
	if isRetryableConnectError(fmt.Errorf("authentication failed")) {
		t.Fatal("expected auth error to be non-retryable")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if sleepWithContext(ctx, time.Second) {
		t.Fatal("sleepWithContext should stop on canceled context")
	}
	if !sleepWithContext(context.Background(), time.Millisecond) {
		t.Fatal("sleepWithContext should succeed for normal delay")
	}

	frame1 := realFacade.combinePayload("msg", 2, 0, []byte("hel"))
	if frame1 != nil {
		t.Fatal("expected first fragment combine to return nil")
	}
	frame2 := realFacade.combinePayload("msg", 2, 1, []byte("lo"))
	if string(frame2) != "hello" {
		t.Fatalf("combined payload = %q, want hello", string(frame2))
	}

	resp := &http.Response{Header: http.Header{}}
	resp.Header.Set(larkws.HeaderHandshakeStatus, "514")
	resp.Header.Set(larkws.HeaderHandshakeMsg, "auth error")
	if got := parseHandshakeError(resp); !strings.Contains(strings.ToLower(got.Error()), "authentication") {
		t.Fatalf("parseHandshakeError() = %v", got)
	}

	resp = &http.Response{Status: "503 Service Unavailable", Header: http.Header{}}
	if got := parseHandshakeError(resp); !strings.Contains(got.Error(), "503 Service Unavailable") {
		t.Fatalf("parseHandshakeError() = %v", got)
	}
}

func TestConvertSDKMessageEvent(t *testing.T) {
	event := newSDKTextEvent("evt-sdk", "om-sdk", "oc-sdk", "hello")
	name := "Tester"
	mentionKey := "@_user_1"
	tenantKey := "tenant-key"
	userID := "user-1"
	unionID := "union-1"
	event.Event.Message.Mentions = []*larkim.MentionEvent{{
		Key:       &mentionKey,
		Name:      &name,
		TenantKey: &tenantKey,
		Id: &larkim.UserId{
			OpenId:  event.Event.Sender.SenderId.OpenId,
			UserId:  &userID,
			UnionId: &unionID,
		},
	}}

	converted, err := convertSDKMessageEvent(event, 1)
	if err != nil {
		t.Fatalf("convertSDKMessageEvent() error = %v", err)
	}
	if converted.EventID != "evt-sdk" || converted.MessageID != "om-sdk" {
		t.Fatalf("unexpected converted IDs: %+v", converted)
	}
	if len(converted.Mentions) != 1 || converted.Mentions[0].Name != "Tester" {
		t.Fatalf("unexpected mentions: %+v", converted.Mentions)
	}

	if _, err := convertSDKMessageEvent(&larkim.P2MessageReceiveV1{}, 2); err == nil {
		t.Fatal("expected nil payload conversion to fail")
	}
}

func TestSDKFacadeSmallHelpers(t *testing.T) {
	if eventIDFromSDK(nil) != "" {
		t.Fatal("eventIDFromSDK(nil) should return empty string")
	}
	if userIDOpen(nil) != nil || userIDUnion(nil) != nil || userIDUser(nil) != nil {
		t.Fatal("userID helpers should return nil for nil input")
	}
	if isRetryableConnectError(nil) {
		t.Fatal("nil error should not be retryable")
	}
}

func TestRealSDKFacadeGetEndpointAndSendText(t *testing.T) {
	var tokenCalls atomic.Int32
	var messageCalls atomic.Int32
	var requestBody string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case larkws.GenEndpointUri:
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"code":0,"msg":"ok","data":{"URL":%q}}`, "ws://example.test/ws?service_id=1&device_id=test")
		case "/open-apis/auth/v3/tenant_access_token/internal":
			tokenCalls.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"code":0,"msg":"ok","tenant_access_token":"token","expire":7200}`)
		case "/open-apis/im/v1/messages":
			messageCalls.Add(1)
			w.Header().Set("Content-Type", "application/json")
			body, _ := io.ReadAll(r.Body)
			requestBody = string(body)
			_, _ = io.WriteString(w, `{"code":0,"msg":"ok","data":{"message_id":"om_test"}}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	facade := &realSDKFacade{
		appID:             "app-id",
		appSecret:         "app-secret",
		baseURL:           server.URL,
		apiClient:         lark.NewClient("app-id", "app-secret", lark.WithOpenBaseUrl(server.URL)),
		reconnectNonce:    0,
		reconnectInterval: time.Second,
		pingInterval:      time.Second,
		fragments:         make(map[string][][]byte),
	}

	endpointURL, _, err := facade.getEndpoint(context.Background())
	if err != nil {
		t.Fatalf("getEndpoint() error = %v", err)
	}
	if !strings.Contains(endpointURL, "service_id=1") {
		t.Fatalf("unexpected endpoint URL: %q", endpointURL)
	}

	if err := facade.SendText(context.Background(), "oc_chat", `{"text":"hello"}`); err != nil {
		t.Fatalf("SendText() error = %v", err)
	}
	if tokenCalls.Load() == 0 || messageCalls.Load() == 0 {
		t.Fatalf("expected token and message endpoints to be called, got token=%d message=%d", tokenCalls.Load(), messageCalls.Load())
	}
	if !strings.Contains(requestBody, `"receive_id":"oc_chat"`) || !strings.Contains(requestBody, `"msg_type":"text"`) {
		t.Fatalf("unexpected send body: %s", requestBody)
	}
}

func TestRealSDKFacadeEndpointAndConnectErrors(t *testing.T) {
	t.Run("endpoint busy returns error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"code":1,"msg":"busy"}`)
		}))
		defer server.Close()

		facade := &realSDKFacade{appID: "app", appSecret: "secret", baseURL: server.URL, fragments: make(map[string][][]byte)}
		if _, _, err := facade.getEndpoint(context.Background()); err == nil {
			t.Fatal("expected system busy endpoint to fail")
		}
	})

	t.Run("endpoint without websocket url returns error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"code":0,"msg":"ok","data":{}}`)
		}))
		defer server.Close()

		facade := &realSDKFacade{appID: "app", appSecret: "secret", baseURL: server.URL, fragments: make(map[string][][]byte)}
		if _, _, err := facade.getEndpoint(context.Background()); err == nil {
			t.Fatal("expected empty websocket url to fail")
		}
	})

	t.Run("websocket handshake auth failure returns error", func(t *testing.T) {
		upgrader := wsconn.Upgrader{}
		var server *httptest.Server
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case larkws.GenEndpointUri:
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintf(w, `{"code":0,"msg":"ok","data":{"URL":%q}}`, websocketURL(server.URL)+"/ws?service_id=9&device_id=test")
			case "/ws":
				w.Header().Set(larkws.HeaderHandshakeStatus, "514")
				w.Header().Set(larkws.HeaderHandshakeMsg, "auth failed")
				w.WriteHeader(http.StatusUnauthorized)
			default:
				_ = upgrader
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		facade := &realSDKFacade{appID: "app", appSecret: "secret", baseURL: server.URL, fragments: make(map[string][][]byte)}
		if err := facade.connect(context.Background()); err == nil {
			t.Fatal("expected handshake failure to fail connect")
		}
	})

	t.Run("writeBinary without conn returns error", func(t *testing.T) {
		facade := &realSDKFacade{}
		if err := facade.writeBinary([]byte("ping")); err == nil {
			t.Fatal("expected writeBinary without conn to fail")
		}
	})
}

func TestRealSDKFacadeStartStopAndPing(t *testing.T) {
	serverConnCh := make(chan *wsconn.Conn, 1)
	receivedClientFrame := make(chan *larkws.Frame, 2)
	messageHandled := make(chan string, 1)

	server := newFacadeWebsocketServer(t, serverConnCh, receivedClientFrame)
	defer server.Close()

	facade := &realSDKFacade{
		appID:             "app-id",
		appSecret:         "app-secret",
		baseURL:           server.URL,
		apiClient:         lark.NewClient("app-id", "app-secret", lark.WithOpenBaseUrl(server.URL)),
		reconnectNonce:    0,
		reconnectInterval: 10 * time.Millisecond,
		pingInterval:      10 * time.Millisecond,
		fragments:         make(map[string][][]byte),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startErrCh := make(chan error, 1)
	readyCh := make(chan struct{}, 1)
	go func() {
		startErrCh <- facade.Start(ctx, sdkFacadeCallbacks{
			OnReady: func() {
				readyCh <- struct{}{}
			},
			OnDisconnected: func(err error) {},
			OnMessage: func(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
				messageHandled <- *event.Event.Message.MessageId
				return nil
			},
		})
	}()

	select {
	case <-readyCh:
	case <-time.After(2 * time.Second):
		t.Fatal("facade did not report ready")
	}

	serverConn := <-serverConnCh
	payload := []byte(`{"schema":"2.0","header":{"event_id":"evt-1","event_type":"im.message.receive_v1","create_time":"1710000000000"},"event":{"sender":{"sender_id":{"open_id":"ou_1"},"sender_type":"user"},"message":{"message_id":"om_1","create_time":"1710000000000","chat_id":"oc_1","chat_type":"p2p","message_type":"text","content":"{\"text\":\"hello\"}"}}}`)
	if err := writeServerEventFrame(serverConn, payload); err != nil {
		t.Fatalf("writeServerEventFrame() error = %v", err)
	}

	select {
	case messageID := <-messageHandled:
		if messageID != "om_1" {
			t.Fatalf("messageID = %q, want om_1", messageID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("dispatcher did not handle websocket event")
	}

	deadline := time.After(2 * time.Second)
	for {
		select {
		case frame := <-receivedClientFrame:
			if larkws.MessageType(larkws.Headers(frame.Headers).GetString(larkws.HeaderType)) == larkws.MessageTypePing {
				goto pingReceived
			}
		case <-deadline:
			t.Fatal("expected ping frame from client")
		}
	}

pingReceived:

	if err := facade.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	cancel()

	select {
	case err := <-startErrCh:
		if err != nil {
			t.Fatalf("Start() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Start() did not exit after Stop()")
	}
}

func TestRealSDKFacadeHandleControlAndDataFrames(t *testing.T) {
	serverConnCh := make(chan *wsconn.Conn, 1)
	receivedClientFrame := make(chan *larkws.Frame, 2)
	server := newFacadeWebsocketServer(t, serverConnCh, receivedClientFrame)
	defer server.Close()

	wsURL := websocketURL(server.URL)
	clientConn, _, err := wsconn.DefaultDialer.Dial(wsURL+"/ws?service_id=7&device_id=test", nil)
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	defer clientConn.Close()
	serverConn := <-serverConnCh
	defer serverConn.Close()

	facade := &realSDKFacade{
		conn:              clientConn,
		serviceID:         7,
		reconnectNonce:    0,
		reconnectInterval: time.Second,
		pingInterval:      time.Second,
		fragments:         make(map[string][][]byte),
	}

	configPayload, _ := json.Marshal(&larkws.ClientConfig{PingInterval: 9})
	controlFrame := &larkws.Frame{Method: int32(larkws.FrameTypeControl), Headers: larkws.Headers{{Key: larkws.HeaderType, Value: string(larkws.MessageTypePong)}}, Payload: configPayload}
	facade.handleControlFrame(controlFrame)
	if facade.pingInterval != 9*time.Second {
		t.Fatalf("pingInterval = %v, want 9s", facade.pingInterval)
	}

	var handled atomic.Int32
	dispatch := dispatcher.NewEventDispatcher("", "").OnP2MessageReceiveV1(func(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
		handled.Add(1)
		return nil
	})
	payload := []byte(`{"schema":"2.0","header":{"event_id":"evt-2","event_type":"im.message.receive_v1","create_time":"1710000000000"},"event":{"sender":{"sender_id":{"open_id":"ou_2"},"sender_type":"user"},"message":{"message_id":"om_2","create_time":"1710000000000","chat_id":"oc_2","chat_type":"p2p","message_type":"text","content":"{\"text\":\"world\"}"}}}`)
	frame := &larkws.Frame{
		Method:  int32(larkws.FrameTypeData),
		Service: 7,
		Headers: larkws.Headers{{Key: larkws.HeaderType, Value: string(larkws.MessageTypeEvent)}, {Key: larkws.HeaderMessageID, Value: "frame-1"}, {Key: larkws.HeaderSum, Value: "1"}, {Key: larkws.HeaderSeq, Value: "0"}},
		Payload: payload,
	}
	if err := facade.handleDataFrame(context.Background(), dispatch, frame); err != nil {
		t.Fatalf("handleDataFrame() error = %v", err)
	}
	if handled.Load() != 1 {
		t.Fatalf("handled = %d, want 1", handled.Load())
	}

	select {
	case responseFrame := <-receivedClientFrame:
		if larkws.Headers(responseFrame.Headers).GetString(larkws.HeaderMessageID) != "frame-1" {
			t.Fatalf("unexpected response frame: %+v", responseFrame)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected response frame from handleDataFrame")
	}

	if err := facade.closeConn(); err != nil {
		t.Fatalf("closeConn() error = %v", err)
	}
	if facade.currentConn() != nil {
		t.Fatal("currentConn() should be nil after closeConn()")
	}
}

func newFacadeWebsocketServer(t *testing.T, serverConnCh chan *wsconn.Conn, receivedClientFrame chan *larkws.Frame) *httptest.Server {
	t.Helper()
	upgrader := wsconn.Upgrader{}

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case larkws.GenEndpointUri:
			wsURL := websocketURL(server.URL) + "/ws?service_id=7&device_id=test"
			fmt.Fprintf(w, `{"code":0,"msg":"ok","data":{"URL":%q}}`, wsURL)
		case "/ws":
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				t.Errorf("Upgrade() error = %v", err)
				return
			}
			serverConnCh <- conn
			go func() {
				for {
					messageType, payload, err := conn.ReadMessage()
					if err != nil {
						return
					}
					if messageType != wsconn.BinaryMessage {
						continue
					}
					frame := &larkws.Frame{}
					if err := frame.Unmarshal(payload); err != nil {
						continue
					}
					receivedClientFrame <- frame
				}
			}()
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	return server
}

func websocketURL(serverURL string) string {
	return "ws" + strings.TrimPrefix(serverURL, "http")
}

func writeServerEventFrame(conn *wsconn.Conn, payload []byte) error {
	frame := &larkws.Frame{
		Method:  int32(larkws.FrameTypeData),
		Service: 7,
		Headers: larkws.Headers{{Key: larkws.HeaderType, Value: string(larkws.MessageTypeEvent)}, {Key: larkws.HeaderMessageID, Value: "server-frame"}, {Key: larkws.HeaderSum, Value: "1"}, {Key: larkws.HeaderSeq, Value: "0"}},
		Payload: payload,
	}
	encoded, err := frame.Marshal()
	if err != nil {
		return err
	}
	return conn.WriteMessage(wsconn.BinaryMessage, encoded)
}
