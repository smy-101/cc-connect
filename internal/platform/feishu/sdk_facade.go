package feishu

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	wsconn "github.com/gorilla/websocket"
	lark "github.com/larksuite/oapi-sdk-go/v3"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"
)

type realSDKFacade struct {
	appID     string
	appSecret string
	baseURL   string

	apiClient *lark.Client

	mu                sync.Mutex
	conn              *wsconn.Conn
	serviceID         int32
	reconnectNonce    int
	reconnectInterval time.Duration
	pingInterval      time.Duration
	fragments         map[string][][]byte
	manualStop        bool
	writeMu           sync.Mutex
}

func newRealSDKFacade(appID, appSecret string) sdkFacade {
	return &realSDKFacade{
		appID:             appID,
		appSecret:         appSecret,
		baseURL:           lark.FeishuBaseUrl,
		apiClient:         lark.NewClient(appID, appSecret, lark.WithOpenBaseUrl(lark.FeishuBaseUrl)),
		reconnectNonce:    30,
		reconnectInterval: 2 * time.Minute,
		pingInterval:      2 * time.Minute,
		fragments:         make(map[string][][]byte),
	}
}

func (f *realSDKFacade) Start(ctx context.Context, callbacks sdkFacadeCallbacks) error {
	if f.appID == "" || f.appSecret == "" {
		return errInvalidCredentials
	}

	dispatcher := dispatcher.NewEventDispatcher("", "").OnP2MessageReceiveV1(func(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
		return callbacks.OnMessage(ctx, event)
	})

	ready := false
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		if err := f.connect(ctx); err != nil {
			if !ready {
				return err
			}
			callbacks.OnDisconnected(err)
			if !isRetryableConnectError(err) || !sleepWithContext(ctx, f.nextReconnectDelay()) {
				if ctx.Err() != nil {
					return nil
				}
				return err
			}
			continue
		}

		callbacks.OnReady()
		ready = true

		pingCtx, stopPing := context.WithCancel(ctx)
		pingDone := make(chan struct{})
		go f.pingLoop(pingCtx, pingDone)

		err := f.readLoop(ctx, dispatcher)
		stopPing()
		<-pingDone

		if err == nil {
			return nil
		}

		callbacks.OnDisconnected(err)
		if !isRetryableConnectError(err) || !sleepWithContext(ctx, f.nextReconnectDelay()) {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
	}
}

func (f *realSDKFacade) Stop(ctx context.Context) error {
	f.mu.Lock()
	f.manualStop = true
	conn := f.conn
	f.conn = nil
	f.mu.Unlock()

	if conn == nil {
		return nil
	}

	_ = conn.WriteControl(wsconn.CloseMessage, wsconn.FormatCloseMessage(wsconn.CloseNormalClosure, "shutdown"), time.Now().Add(time.Second))
	return conn.Close()
}

func (f *realSDKFacade) SendText(ctx context.Context, chatID, content string) error {
	body := larkim.NewCreateMessageReqBodyBuilder().
		ReceiveId(chatID).
		MsgType("text").
		Content(content).
		Uuid(uuid.NewString()).
		Build()

	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType("chat_id").
		Body(body).
		Build()

	resp, err := f.apiClient.Im.V1.Message.Create(ctx, req)
	if err != nil {
		return err
	}
	if !resp.Success() {
		return fmt.Errorf("feishu send text failed: code=%d msg=%s", resp.Code, resp.Msg)
	}

	return nil
}

func (f *realSDKFacade) connect(ctx context.Context) error {
	endpointURL, config, err := f.getEndpoint(ctx)
	if err != nil {
		return err
	}
	if config != nil {
		f.applyConfig(config)
	}

	parsed, err := url.Parse(endpointURL)
	if err != nil {
		return err
	}

	serviceID, _ := strconv.ParseInt(parsed.Query().Get(larkws.ServiceID), 10, 32)
	conn, resp, err := wsconn.DefaultDialer.DialContext(ctx, endpointURL, nil)
	if err != nil {
		if resp != nil && resp.StatusCode != http.StatusSwitchingProtocols {
			return parseHandshakeError(resp)
		}
		return err
	}
	if resp != nil && resp.StatusCode != http.StatusSwitchingProtocols {
		_ = conn.Close()
		return parseHandshakeError(resp)
	}

	f.mu.Lock()
	if f.conn != nil {
		_ = f.conn.Close()
	}
	f.conn = conn
	f.serviceID = int32(serviceID)
	f.manualStop = false
	f.mu.Unlock()

	return nil
}

func (f *realSDKFacade) readLoop(ctx context.Context, dispatcher *dispatcher.EventDispatcher) error {
	for {
		if err := ctx.Err(); err != nil {
			_ = f.Stop(context.Background())
			return nil
		}

		conn := f.currentConn()
		if conn == nil {
			if f.isManualStop() {
				return nil
			}
			return errors.New("connection is closed")
		}

		messageType, payload, err := conn.ReadMessage()
		if err != nil {
			_ = f.closeConn()
			if f.isManualStop() || ctx.Err() != nil {
				return nil
			}
			return err
		}

		if messageType != wsconn.BinaryMessage {
			continue
		}

		frame := &larkws.Frame{}
		if err := frame.Unmarshal(payload); err != nil {
			continue
		}

		switch larkws.FrameType(frame.Method) {
		case larkws.FrameTypeControl:
			f.handleControlFrame(frame)
		case larkws.FrameTypeData:
			if err := f.handleDataFrame(ctx, dispatcher, frame); err != nil {
				return err
			}
		}
	}
}

func (f *realSDKFacade) pingLoop(ctx context.Context, done chan struct{}) {
	defer close(done)
	ticker := time.NewTicker(f.pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			frame := larkws.NewPingFrame(f.currentServiceID())
			payload, err := frame.Marshal()
			if err != nil {
				continue
			}
			_ = f.writeBinary(payload)
		}
	}
}

func (f *realSDKFacade) handleControlFrame(frame *larkws.Frame) {
	headers := larkws.Headers(frame.Headers)
	if larkws.MessageType(headers.GetString(larkws.HeaderType)) != larkws.MessageTypePong {
		return
	}
	if len(frame.Payload) == 0 {
		return
	}

	config := &larkws.ClientConfig{}
	if err := json.Unmarshal(frame.Payload, config); err != nil {
		return
	}
	f.applyConfig(config)
}

func (f *realSDKFacade) handleDataFrame(ctx context.Context, dispatcher *dispatcher.EventDispatcher, frame *larkws.Frame) error {
	headers := larkws.Headers(frame.Headers)
	sum := headers.GetInt(larkws.HeaderSum)
	seq := headers.GetInt(larkws.HeaderSeq)
	messageID := headers.GetString(larkws.HeaderMessageID)
	payload := frame.Payload
	if sum > 1 {
		payload = f.combinePayload(messageID, sum, seq, payload)
		if payload == nil {
			return nil
		}
	}

	if larkws.MessageType(headers.GetString(larkws.HeaderType)) != larkws.MessageTypeEvent {
		return nil
	}

	response := larkws.NewResponseByCode(http.StatusOK)
	rsp, err := dispatcher.Do(ctx, payload)
	if err != nil {
		response = larkws.NewResponseByCode(http.StatusInternalServerError)
	} else if rsp != nil {
		data, marshalErr := json.Marshal(rsp)
		if marshalErr != nil {
			response = larkws.NewResponseByCode(http.StatusInternalServerError)
		} else {
			response.Data = data
		}
	}

	encoded, marshalErr := json.Marshal(response)
	if marshalErr != nil {
		return marshalErr
	}
	frame.Payload = encoded
	binary, marshalErr := frame.Marshal()
	if marshalErr != nil {
		return marshalErr
	}

	return f.writeBinary(binary)
}

func (f *realSDKFacade) combinePayload(messageID string, sum, seq int, payload []byte) []byte {
	f.mu.Lock()
	defer f.mu.Unlock()

	buf, ok := f.fragments[messageID]
	if !ok {
		buf = make([][]byte, sum)
	}
	buf[seq] = payload
	f.fragments[messageID] = buf

	capacity := 0
	for _, fragment := range buf {
		if len(fragment) == 0 {
			return nil
		}
		capacity += len(fragment)
	}

	combined := make([]byte, 0, capacity)
	for _, fragment := range buf {
		combined = append(combined, fragment...)
	}
	delete(f.fragments, messageID)
	return combined
}

func (f *realSDKFacade) getEndpoint(ctx context.Context) (string, *larkws.ClientConfig, error) {
	body := map[string]string{
		"AppID":     f.appID,
		"AppSecret": f.appSecret,
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return "", nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.baseURL+larkws.GenEndpointUri, bytes.NewBuffer(encoded))
	if err != nil {
		return "", nil, err
	}
	req.Header.Add("locale", "zh")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("feishu endpoint request failed: status=%d", resp.StatusCode)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, err
	}

	endpoint := &larkws.EndpointResp{}
	if err := json.Unmarshal(raw, endpoint); err != nil {
		return "", nil, err
	}

	switch endpoint.Code {
	case larkws.OK:
	case larkws.SystemBusy:
		return "", nil, fmt.Errorf("feishu endpoint unavailable: system busy")
	default:
		return "", nil, fmt.Errorf("feishu endpoint failed: code=%d msg=%s", endpoint.Code, endpoint.Msg)
	}

	if endpoint.Data == nil || endpoint.Data.Url == "" {
		return "", nil, errors.New("feishu endpoint returned empty websocket url")
	}

	return endpoint.Data.Url, endpoint.Data.ClientConfig, nil
}

func (f *realSDKFacade) applyConfig(config *larkws.ClientConfig) {
	if config == nil {
		return
	}
	if config.ReconnectNonce > 0 {
		f.reconnectNonce = config.ReconnectNonce
	}
	if config.ReconnectInterval > 0 {
		f.reconnectInterval = time.Duration(config.ReconnectInterval) * time.Second
	}
	if config.PingInterval > 0 {
		f.pingInterval = time.Duration(config.PingInterval) * time.Second
	}
}

func (f *realSDKFacade) nextReconnectDelay() time.Duration {
	if f.reconnectNonce <= 0 {
		return f.reconnectInterval
	}
	return time.Duration(rand.Intn(f.reconnectNonce*1000))*time.Millisecond + f.reconnectInterval
}

func (f *realSDKFacade) currentConn() *wsconn.Conn {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.conn
}

func (f *realSDKFacade) currentServiceID() int32 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.serviceID
}

func (f *realSDKFacade) isManualStop() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.manualStop
}

func (f *realSDKFacade) closeConn() error {
	f.mu.Lock()
	conn := f.conn
	f.conn = nil
	f.mu.Unlock()
	if conn == nil {
		return nil
	}
	return conn.Close()
}

func (f *realSDKFacade) writeBinary(payload []byte) error {
	conn := f.currentConn()
	if conn == nil {
		return errors.New("connection is closed")
	}
	f.writeMu.Lock()
	defer f.writeMu.Unlock()
	return conn.WriteMessage(wsconn.BinaryMessage, payload)
}

func convertSDKMessageEvent(event *larkim.P2MessageReceiveV1, sequence int64) (*MessageReceiveEvent, error) {
	if event == nil || event.Event == nil || event.Event.Message == nil {
		return nil, errors.New("sdk event is nil")
	}

	message := event.Event.Message
	sender := event.Event.Sender
	converted := &MessageReceiveEvent{
		EventID:     eventIDFromSDK(event),
		MessageID:   stringValue(message.MessageId),
		MessageType: stringValue(message.MessageType),
		Content:     stringValue(message.Content),
		ChatID:      stringValue(message.ChatId),
		ChatType:    stringValue(message.ChatType),
		CreateTime:  NewEventParser().parseTimestamp(stringValue(message.CreateTime)),
		RawEvent:    event,
	}
	if converted.EventID == "" {
		converted.EventID = fmt.Sprintf("sdk-event-%d", sequence)
	}

	if sender != nil {
		converted.Sender = SenderInfo{
			OpenID:     stringValue(userIDOpen(sender.SenderId)),
			UnionID:    stringValue(userIDUnion(sender.SenderId)),
			UserID:     stringValue(userIDUser(sender.SenderId)),
			SenderType: stringValue(sender.SenderType),
		}
	}

	if len(message.Mentions) > 0 {
		converted.Mentions = make([]MentionInfo, 0, len(message.Mentions))
		for _, mention := range message.Mentions {
			converted.Mentions = append(converted.Mentions, MentionInfo{
				Key:       stringValue(mention.Key),
				OpenID:    stringValue(userIDOpen(mention.Id)),
				UnionID:   stringValue(userIDUnion(mention.Id)),
				UserID:    stringValue(userIDUser(mention.Id)),
				Name:      stringValue(mention.Name),
				TenantKey: stringValue(mention.TenantKey),
			})
		}
	}

	if converted.MessageID == "" {
		return nil, errors.New("missing message id in sdk event")
	}

	return converted, nil
}

func parseHandshakeError(resp *http.Response) error {
	if resp == nil {
		return errors.New("feishu websocket handshake failed")
	}
	status := resp.Header.Get(larkws.HeaderHandshakeStatus)
	message := resp.Header.Get(larkws.HeaderHandshakeMsg)
	if status == strconv.Itoa(larkws.AuthFailed) || resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("feishu websocket authentication failed: %s", message)
	}
	if message == "" {
		message = resp.Status
	}
	return fmt.Errorf("feishu websocket handshake failed: %s", message)
}

func isRetryableConnectError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return !strings.Contains(message, "auth") && !strings.Contains(message, "credential")
}

func sleepWithContext(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		d = time.Second
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func eventIDFromSDK(event *larkim.P2MessageReceiveV1) string {
	if event == nil || event.EventV2Base == nil || event.EventV2Base.Header == nil {
		return ""
	}
	return event.EventV2Base.Header.EventID
}

func userIDOpen(value *larkim.UserId) *string {
	if value == nil {
		return nil
	}
	return value.OpenId
}

func userIDUnion(value *larkim.UserId) *string {
	if value == nil {
		return nil
	}
	return value.UnionId
}

func userIDUser(value *larkim.UserId) *string {
	if value == nil {
		return nil
	}
	return value.UserId
}
