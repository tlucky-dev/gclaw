package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"gclaw/pkg/types"
)

// Config 飞书配置
type Config struct {
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
	EncryptKey string `json:"encrypt_key,omitempty"` // 可选，用于消息加密
	VerificationToken string `json:"verification_token,omitempty"` // 可选，用于事件订阅验证
}

// Adapter 飞书适配器
type Adapter struct {
	config      Config
	accessToken string
	tokenExpiry time.Time
	mu          sync.RWMutex
	httpClient  *http.Client
	messageChan chan types.Message
	callbacks   map[string]func(types.Message)
	cbMu        sync.RWMutex
}

// EventResponse 飞书事件响应
type EventResponse struct {
	Challenge string `json:"challenge,omitempty"`
	Code      int    `json:"code,omitempty"`
	Msg       string `json:"msg,omitempty"`
}

// WebhookRequest 飞书 webhook 请求结构
type WebhookRequest struct {
	Challenge string          `json:"challenge,omitempty"`
	Token     string          `json:"token,omitempty"`
	Type      string          `json:"type,omitempty"`
	Event     map[string]interface{} `json:"event,omitempty"`
	Header    map[string]interface{} `json:"header,omitempty"`
}

// NewAdapter 创建新的飞书适配器
func NewAdapter(cfg Config) *Adapter {
	return &Adapter{
		config:      cfg,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		messageChan: make(chan types.Message, 100),
		callbacks:   make(map[string]func(types.Message)),
	}
}

// GetAccessToken 获取访问令牌
func (a *Adapter) GetAccessToken() (string, error) {
	a.mu.RLock()
	if a.accessToken != "" && time.Now().Before(a.tokenExpiry) {
		token := a.accessToken
		a.mu.RUnlock()
		return token, nil
	}
	a.mu.RUnlock()

	a.mu.Lock()
	defer a.mu.Unlock()

	// 再次检查（双重检查锁定）
	if a.accessToken != "" && time.Now().Before(a.tokenExpiry) {
		return a.accessToken, nil
	}

	url := "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal"
	reqBody := map[string]string{
		"app_id":     a.config.AppID,
		"app_secret": a.config.AppSecret,
	}

	body, _ := json.Marshal(reqBody)
	resp, err := a.httpClient.Post(url, "application/json", strings.NewReader(string(body)))
	if err != nil {
		return "", fmt.Errorf("获取 access token 失败：%w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Code              int    `json:"code"`
		Msg               string `json:"msg"`
		TenantAccessToken string `json:"tenant_access_token"`
		Expire            int    `json:"expire"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("解析响应失败：%w", err)
	}

	if result.Code != 0 {
		return "", fmt.Errorf("飞书 API 错误：%d - %s", result.Code, result.Msg)
	}

	a.accessToken = result.TenantAccessToken
	a.tokenExpiry = time.Now().Add(time.Duration(result.Expire-300) * time.Second) // 提前 5 分钟过期

	return a.accessToken, nil
}

// StartMessageChannel 启动消息接收通道
func (a *Adapter) StartMessageChannel(ctx context.Context) <-chan types.Message {
	go func() {
		<-ctx.Done()
		close(a.messageChan)
	}()
	return a.messageChan
}

// HandleWebhook 处理飞书 webhook 回调
func (a *Adapter) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "读取请求体失败", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req WebhookRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "解析 JSON 失败", http.StatusBadRequest)
		return
	}

	// 处理 URL 验证挑战
	if req.Type == "url_verification" && req.Challenge != "" {
		resp := EventResponse{Challenge: req.Challenge}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// 验证事件订阅 token
	if a.config.VerificationToken != "" && req.Token != "" {
		if req.Token != a.config.VerificationToken {
			http.Error(w, "无效的 verification token", http.StatusForbidden)
			return
		}
	}

	// 解析事件
	if req.Event != nil {
		msg := a.parseEvent(req.Event)
		if msg != nil {
			select {
			case a.messageChan <- *msg:
			default:
				fmt.Fprintf(os.Stderr, "消息队列已满，丢弃消息\n")
			}
		}
	}

	// 返回成功响应
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(EventResponse{Code: 0, Msg: "success"})
}

// parseEvent 解析飞书事件为统一消息格式
func (a *Adapter) parseEvent(event map[string]interface{}) *types.Message {
	eventType, ok := event["type"].(string)
	if !ok {
		return nil
	}

	// 目前主要支持消息事件
	if eventType != "im.message.receive_v1" {
		return nil
	}

	message, ok := event["message"].(map[string]interface{})
	if !ok {
		return nil
	}

	sender, ok := event["sender"].(map[string]interface{})
	if !ok {
		return nil
	}

	senderID := ""
	if id, ok := sender["sender_id"].(map[string]interface{}); ok {
		if uid, ok := id["open_id"].(string); ok {
			senderID = uid
		} else if uid, ok := id["user_id"].(string); ok {
			senderID = uid
		}
	}

	content := ""
	if c, ok := message["content"].(string); ok {
		// 飞书消息内容是 JSON 字符串，需要解析
		var contentMap map[string]interface{}
		if err := json.Unmarshal([]byte(c), &contentMap); err == nil {
			if text, ok := contentMap["text"].(string); ok {
				content = text
			}
		} else {
			content = c
		}
	}

	messageID := ""
	if id, ok := message["message_id"].(string); ok {
		messageID = id
	}

	conversationID := ""
	if id, ok := event["chat_id"].(string); ok {
		conversationID = id
	}

	return &types.Message{
		ID:             messageID,
		ConversationID: conversationID,
		SenderID:       senderID,
		Content:        content,
		Timestamp:      time.Now(),
		Metadata: map[string]interface{}{
			"source":    "feishu",
			"raw_event": event,
		},
	}
}

// SendMessage 发送消息到飞书
func (a *Adapter) SendMessage(conversationID, content string) error {
	token, err := a.GetAccessToken()
	if err != nil {
		return err
	}

	url := "https://open.feishu.cn/open-apis/im/v1/messages"
	
	// 构建消息内容（文本类型）
	contentJSON, _ := json.Marshal(map[string]string{
		"text": content,
	})

	reqBody := map[string]interface{}{
		"receive_id": conversationID,
		"msg_type":   "text",
		"content":    string(contentJSON),
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", url, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("创建请求失败：%w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("发送消息失败：%w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("解析响应失败：%w", err)
	}

	if result.Code != 0 {
		return fmt.Errorf("飞书 API 错误：%d - %s", result.Code, result.Msg)
	}

	return nil
}

// RegisterCallback 注册消息回调
func (a *Adapter) RegisterCallback(id string, fn func(types.Message)) {
	a.cbMu.Lock()
	defer a.cbMu.Unlock()
	a.callbacks[id] = fn
}

// UnregisterCallback 注销消息回调
func (a *Adapter) UnregisterCallback(id string) {
	a.cbMu.Lock()
	defer a.cbMu.Unlock()
	delete(a.callbacks, id)
}

// GetConfig 获取配置
func (a *Adapter) GetConfig() Config {
	return a.config
}
