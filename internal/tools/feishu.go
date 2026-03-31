package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/go-resty/resty/v2"
)

// FeishuTool 飞书工具集
type FeishuTool struct {
	BaseTool
	client *resty.Client
}

// FeishuConfig 飞书配置
type FeishuConfig struct {
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
}

// NewFeishuTool 创建飞书工具
func NewFeishuTool() *FeishuTool {
	client := resty.New().
		SetBaseURL("https://open.feishu.cn/open-apis").
		SetTimeout(30 * time.Second).
		SetRetryCount(3)

	tool := &FeishuTool{
		BaseTool: BaseTool{
			NameVal:        "feishu",
			DescriptionVal: "Interact with Feishu (send messages, get user info, etc.)",
			ParametersVal: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"action": map[string]interface{}{
						"type":        "string",
						"description": "Action to perform: send_message, get_user_info, create_chat",
						"enum":        []string{"send_message", "get_user_info", "create_chat"},
					},
					"params": map[string]interface{}{
						"type":        "object",
						"description": "Parameters for the action",
					},
				},
				"required": []string{"action", "params"},
			},
		},
		client: client,
	}

	return tool
}

// Execute 执行飞书操作
func (t *FeishuTool) Execute(args map[string]interface{}) (string, error) {
	action, ok := args["action"].(string)
	if !ok {
		return "", fmt.Errorf("invalid action argument")
	}

	params, ok := args["params"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid params argument")
	}

	// 获取 tenant_access_token
	token, err := t.getTenantAccessToken()
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %v", err)
	}

	ctx := context.Background()

	switch action {
	case "send_message":
		return t.sendMessage(ctx, token, params)
	case "get_user_info":
		return t.getUserInfo(ctx, token, params)
	case "create_chat":
		return t.createChat(ctx, token, params)
	default:
		return "", fmt.Errorf("unknown action: %s", action)
	}
}

// getTenantAccessToken 获取 tenant_access_token
func (t *FeishuTool) getTenantAccessToken() (string, error) {
	appID := os.Getenv("FEISHU_APP_ID")
	appSecret := os.Getenv("FEISHU_APP_SECRET")

	if appID == "" || appSecret == "" {
		return "", fmt.Errorf("FEISHU_APP_ID or FEISHU_APP_SECRET not set")
	}

	type TokenResponse struct {
		Code              int    `json:"code"`
		Msg               string `json:"msg"`
		TenantAccessToken string `json:"tenant_access_token"`
		Expire            int    `json:"expire"`
	}

	resp, err := t.client.R().
		SetBody(map[string]string{
			"app_id":     appID,
			"app_secret": appSecret,
		}).
		SetResult(&TokenResponse{}).
		Post("/auth/v3/tenant_access_token/internal")

	if err != nil {
		return "", err
	}

	var result TokenResponse
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return "", err
	}

	if result.Code != 0 {
		return "", fmt.Errorf("failed to get token: %s", result.Msg)
	}

	return result.TenantAccessToken, nil
}

// sendMessage 发送消息
func (t *FeishuTool) sendMessage(ctx context.Context, token string, params map[string]interface{}) (string, error) {
	receiveID, ok := params["receive_id"].(string)
	if !ok {
		return "", fmt.Errorf("receive_id is required")
	}

	msgType, ok := params["msg_type"].(string)
	if !ok {
		msgType = "text"
	}

	content, err := json.Marshal(params["content"])
	if err != nil {
		return "", fmt.Errorf("failed to marshal content: %v", err)
	}

	type MessageResponse struct {
		Code      int    `json:"code"`
		Msg       string `json:"msg"`
		MessageID string `json:"data"`
	}

	resp, err := t.client.R().
		SetContext(ctx).
		SetHeader("Authorization", "Bearer "+token).
		SetQueryParams(map[string]string{
			"receive_id": receiveID,
		}).
		SetBody(map[string]interface{}{
			"receive_id": receiveID,
			"msg_type":   msgType,
			"content":    string(content),
		}).
		SetResult(&MessageResponse{}).
		Post("/im/v1/messages")

	if err != nil {
		return "", err
	}

	var result MessageResponse
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return "", err
	}

	if result.Code != 0 {
		return "", fmt.Errorf("failed to send message: %s", result.Msg)
	}

	return fmt.Sprintf("Message sent successfully, message_id: %s", result.MessageID), nil
}

// getUserInfo 获取用户信息
func (t *FeishuTool) getUserInfo(ctx context.Context, token string, params map[string]interface{}) (string, error) {
	userID, ok := params["user_id"].(string)
	if !ok {
		return "", fmt.Errorf("user_id is required")
	}

	type UserInfo struct {
		UserID   string `json:"user_id"`
		Name     string `json:"name"`
		EnName   string `json:"en_name"`
		NickName string `json:"nick_name"`
		Mobile   string `json:"mobile"`
		Email    string `json:"email"`
	}

	type UserResponse struct {
		Code int      `json:"code"`
		Msg  string   `json:"msg"`
		Data UserInfo `json:"data"`
	}

	resp, err := t.client.R().
		SetContext(ctx).
		SetHeader("Authorization", "Bearer "+token).
		SetResult(&UserResponse{}).
		Get(fmt.Sprintf("/contact/v1/users/%s", userID))

	if err != nil {
		return "", err
	}

	var result UserResponse
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return "", err
	}

	if result.Code != 0 {
		return "", fmt.Errorf("failed to get user info: %s", result.Msg)
	}

	data, _ := json.MarshalIndent(result.Data, "", "  ")
	return string(data), nil
}

// createChat 创建群聊
func (t *FeishuTool) createChat(ctx context.Context, token string, params map[string]interface{}) (string, error) {
	name, ok := params["name"].(string)
	if !ok {
		return "", fmt.Errorf("name is required")
	}

	chatType := "group"
	if t, ok := params["chat_type"].(string); ok {
		chatType = t
	}

	type ChatResponse struct {
		Code    int    `json:"code"`
		Msg     string `json:"msg"`
		ChatID  string `json:"data"`
	}

	resp, err := t.client.R().
		SetContext(ctx).
		SetHeader("Authorization", "Bearer "+token).
		SetBody(map[string]interface{}{
			"name":     name,
			"chat_type": chatType,
		}).
		SetResult(&ChatResponse{}).
		Post("/im/v1/chats")

	if err != nil {
		return "", err
	}

	var result ChatResponse
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return "", err
	}

	if result.Code != 0 {
		return "", fmt.Errorf("failed to create chat: %s", result.Msg)
	}

	return fmt.Sprintf("Chat created successfully, chat_id: %s", result.ChatID), nil
}
