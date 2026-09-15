package chat

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"study-gin-clerk/internal/auth"
	"study-gin-clerk/internal/types"
	"study-gin-clerk/internal/user"
)

type Handler struct {
	chatService *Service
	userService *user.Service
}

func NewHandler(chatService *Service, userService *user.Service) *Handler {
	return &Handler{
		chatService: chatService,
		userService: userService,
	}
}

// ListModels godoc
// @Summary 利用可能モデル一覧取得 (動的取得 & キャッシュ)
// @Description 認証済みユーザーが登録した API キーに基づき、プロバイダから動的に取得したモデル一覧を返却します。15分間のインメモリキャッシュ付きです。
// @Tags ai_models
// @Security BearerAuth
// @Produce json
// @Param refresh query bool false "キャッシュをバイパスして強制再取得するか"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]string "未認証"
// @Failure 500 {object} map[string]string "サーバーエラー"
// @Router /api/v1/ai/models [get]
func (h *Handler) ListModels(c *gin.Context) {
	userID := auth.MustGetUserID(c)
	refresh := c.Query("refresh") == "true"

	keys, err := h.userService.GetDecryptedKeys(c.Request.Context(), userID)
	if err != nil {
		slog.Error("failed to retrieve user keys", "error", err, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve models"})
		return
	}

	models, activeProviders, err := h.chatService.GetAvailableModels(c.Request.Context(), userID, keys, refresh)
	if err != nil {
		slog.Error("failed to retrieve models", "error", err, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve models"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"active_providers": activeProviders,
		"models":           models,
	})
}

type CreateChatRequest struct {
	Title string `json:"title" binding:"max=255"`
}

// CreateChat godoc
// @Summary チャット新規作成
// @Description 新しい会話セッションを作成します（モデル指定は不要でメッセージ送信時に選択します）
// @Tags chats
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateChatRequest false "チャット設定 (タイトル, 最大255文字)"
// @Success 201 {object} Chat
// @Failure 400 {object} map[string]string "不正なリクエスト"
// @Failure 401 {object} map[string]string "未認証"
// @Failure 403 {object} map[string]string "認可エラー・トークン不正"
// @Failure 500 {object} map[string]string "サーバーエラー"
// @Router /api/v1/chats [post]
func (h *Handler) CreateChat(c *gin.Context) {
	userID := auth.MustGetUserID(c)

	var req CreateChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	res, err := h.chatService.CreateChat(c.Request.Context(), userID, req.Title)
	if err != nil {
		slog.Error("failed to create chat", "error", err, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create chat"})
		return
	}

	c.JSON(http.StatusCreated, res)
}

// ListChats godoc
// @Summary 自分のチャット一覧取得
// @Description 認証済みユーザーが作成したチャット一覧を取得します
// @Tags chats
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string][]Chat
// @Failure 401 {object} map[string]string "未認証"
// @Failure 403 {object} map[string]string "認可エラー・トークン不正"
// @Failure 500 {object} map[string]string "サーバーエラー"
// @Router /api/v1/chats [get]
func (h *Handler) ListChats(c *gin.Context) {
	userID := auth.MustGetUserID(c)

	chats, err := h.chatService.ListChats(c.Request.Context(), userID)
	if err != nil {
		slog.Error("failed to list chats", "error", err, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list chats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"chats": chats})
}

// GetChat godoc
// @Summary チャット詳細・メッセージ履歴取得
// @Description 指定されたIDのチャット詳細と過去メッセージ履歴を取得します
// @Tags chats
// @Security BearerAuth
// @Produce json
// @Param id path int true "Chat ID"
// @Success 200 {object} Chat
// @Failure 400 {object} map[string]string "不正なID"
// @Failure 401 {object} map[string]string "未認証"
// @Failure 403 {object} map[string]string "認可エラー・トークン不正"
// @Failure 404 {object} map[string]string "チャットが見つからない"
// @Router /api/v1/chats/{id} [get]
func (h *Handler) GetChat(c *gin.Context) {
	userID := auth.MustGetUserID(c)

	chatID, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chat id"})
		return
	}

	res, err := h.chatService.GetChat(c.Request.Context(), chatID, userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "chat not found"})
			return
		}
		slog.Error("failed to get chat", "error", err, "user_id", userID, "chat_id", chatID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get chat"})
		return
	}

	c.JSON(http.StatusOK, res)
}

type SendMessageRequest struct {
	Content  string         `json:"content" binding:"required,max=30000"`
	Provider types.Provider `json:"provider" binding:"required"`
	Model    string         `json:"model" binding:"required,max=100"`
}

func (h *Handler) parseSendMessageRequest(c *gin.Context) (uint, *SendMessageRequest, bool) {
	chatID, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chat id"})
		return 0, nil, false
	}

	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content (max 30000 chars), provider, and model are required"})
		return 0, nil, false
	}

	req.Content = strings.TrimSpace(req.Content)
	if req.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content cannot be empty or whitespace only"})
		return 0, nil, false
	}
	req.Model = strings.TrimSpace(req.Model)
	if req.Model == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model is required"})
		return 0, nil, false
	}

	provider, err := types.ParseProvider(string(req.Provider))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return 0, nil, false
	}
	req.Provider = provider

	return chatID, &req, true
}

// SendMessage godoc
// @Summary メッセージ送信＆AI返答一括生成
// @Description ユーザーメッセージを送信し、指定されたプロバイダ・モデルを使ってAI返答を一括生成して保存します（一括JSON）
// @Tags chats
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Chat ID"
// @Param request body SendMessageRequest true "メッセージ内容・プロバイダ・モデル"
// @Success 200 {object} map[string]Message
// @Failure 400 {object} map[string]string "不正なリクエスト"
// @Failure 401 {object} map[string]string "未認証"
// @Failure 404 {object} map[string]string "チャットが見つからない"
// @Failure 502 {object} map[string]string "AIプロバイダ通信エラー"
// @Failure 500 {object} map[string]string "サーバーエラー"
// @Router /api/v1/chats/{id}/messages [post]
func (h *Handler) SendMessage(c *gin.Context) {
	userID := auth.MustGetUserID(c)

	chatID, req, ok := h.parseSendMessageRequest(c)
	if !ok {
		return
	}

	apiKey, err := h.userService.GetDecryptedKey(c.Request.Context(), userID, req.Provider)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "API key is not registered for provider: " + string(req.Provider)})
		return
	}

	userProfile, _ := h.userService.GetProfile(c.Request.Context(), userID)
	systemPrompt := ""
	if userProfile != nil {
		systemPrompt = userProfile.SystemPrompt
	}

	userMsg, aiMsg, err := h.chatService.SendMessage(
		c.Request.Context(),
		chatID,
		userID,
		req.Content,
		req.Provider,
		req.Model,
		apiKey,
		systemPrompt,
	)
	if err != nil {
		handleChatError(c, err, userID, chatID)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_message":      userMsg,
		"assistant_message": aiMsg,
	})
}

// StreamMessage godoc
// @Summary メッセージ送信＆AI返答リアルタイムストリーミング (SSE)
// @Description ユーザーメッセージを送信し、指定されたプロバイダ・モデルでAI返答を Server-Sent Events (SSE) で1文字ずつリアルタイム受信します。完了時に全文が自動保存されます。
// @Tags chats
// @Security BearerAuth
// @Accept json
// @Produce text/event-stream
// @Param id path int true "Chat ID"
// @Param request body SendMessageRequest true "メッセージ内容・プロバイダ・モデル"
// @Success 200 {string} string "text/event-stream"
// @Failure 400 {object} map[string]string "不正なリクエスト"
// @Failure 401 {object} map[string]string "未認証"
// @Failure 404 {object} map[string]string "チャットが見つからない"
// @Failure 502 {object} map[string]string "AIプロバイダ通信エラー"
// @Failure 500 {object} map[string]string "サーバーエラー"
// @Router /api/v1/chats/{id}/messages/stream [post]
func (h *Handler) StreamMessage(c *gin.Context) {
	userID := auth.MustGetUserID(c)

	chatID, req, ok := h.parseSendMessageRequest(c)
	if !ok {
		return
	}

	apiKey, err := h.userService.GetDecryptedKey(c.Request.Context(), userID, req.Provider)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "API key is not registered for provider: " + string(req.Provider)})
		return
	}

	userProfile, _ := h.userService.GetProfile(c.Request.Context(), userID)
	systemPrompt := ""
	if userProfile != nil {
		systemPrompt = userProfile.SystemPrompt
	}

	streamResult, err := h.chatService.StreamMessage(
		c.Request.Context(),
		chatID,
		userID,
		req.Content,
		req.Provider,
		req.Model,
		apiKey,
		systemPrompt,
	)
	if err != nil {
		handleChatError(c, err, userID, chatID)
		return
	}

	// SSE ヘッダーを設定
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")
	c.Writer.Header().Set("X-Accel-Buffering", "no") // Nginx のバッファリングを無効化
	c.Writer.Flush()

	// 1. ユーザーメッセージを通知
	c.SSEvent("user_message", streamResult.UserMessage)
	c.Writer.Flush()

	// 2. チャンクをリアルタイムに送信
	var fullText strings.Builder
	for chunk := range streamResult.TextStream.TextStream() {
		fullText.WriteString(chunk)
		c.SSEvent("chunk", gin.H{"chunk": chunk})
		c.Writer.Flush()
	}

	// 3. ストリームエラーの確認 (クライアント切断時はノイズログを抑制)
	if err := streamResult.TextStream.Err(); err != nil {
		if errors.Is(err, context.Canceled) {
			slog.Info("streaming client disconnected", "user_id", userID, "chat_id", chatID)
			return
		}
		slog.Error("stream error from AI provider", "error", err, "user_id", userID, "chat_id", chatID)
		c.SSEvent("error", gin.H{"error": err.Error()})
		c.Writer.Flush()
		return
	}

	// 4. DB に完成した全文を保存
	aiMsg, err := streamResult.OnComplete(fullText.String())
	if err != nil {
		slog.Error("failed to persist assistant message", "error", err, "user_id", userID, "chat_id", chatID)
		c.SSEvent("error", gin.H{"error": "failed to persist assistant message"})
		c.Writer.Flush()
		return
	}

	// 5. 完了シグナルを通知
	c.SSEvent("done", aiMsg)
	c.Writer.Flush()
}

func handleChatError(c *gin.Context, err error, userID string, chatID uint) {
	if errors.Is(err, ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "chat not found"})
		return
	}
	if errors.Is(err, ErrAPIKeyNotConfigured) || errors.Is(err, ErrValidationFailed) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if errors.Is(err, ErrAIProvider) {
		slog.Error("AI provider error", "error", err, "user_id", userID, "chat_id", chatID)
		errMsg := err.Error()
		if trimmed := strings.TrimPrefix(errMsg, ErrAIProvider.Error()+": "); trimmed != errMsg {
			errMsg = trimmed
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": errMsg})
		return
	}

	slog.Error("internal chat error", "error", err, "user_id", userID, "chat_id", chatID)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process chat request"})
}

func parseUintParam(c *gin.Context, paramName string) (uint, error) {
	val := c.Param(paramName)
	parsed, err := strconv.ParseUint(val, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint(parsed), nil
}
