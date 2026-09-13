package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"study-gin-clerk/internal/auth"
	"study-gin-clerk/internal/model"
	"study-gin-clerk/internal/service"
)

type ChatHandler struct {
	chatService *service.ChatService
}

func NewChatHandler(chatService *service.ChatService) *ChatHandler {
	return &ChatHandler{chatService: chatService}
}

type CreateChatRequest struct {
	Title string `json:"title"`
}

// CreateChat godoc
// @Summary チャット新規作成
// @Description 新しい会話セッションを作成します（モデル指定は不要でメッセージ送信時に選択します）
// @Tags chats
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateChatRequest false "チャット設定 (タイトル)"
// @Success 201 {object} model.Chat
// @Failure 401 {object} map[string]string "未認証"
// @Failure 403 {object} map[string]string "認可エラー・トークン不正"
// @Failure 500 {object} map[string]string "サーバーエラー"
// @Router /api/v1/chats [post]
func (h *ChatHandler) CreateChat(c *gin.Context) {
	userID := auth.MustGetUserID(c)

	var req CreateChatRequest
	_ = c.ShouldBindJSON(&req)

	chat, err := h.chatService.CreateChat(c.Request.Context(), userID, req.Title)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create chat: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, chat)
}

// ListChats godoc
// @Summary 自分のチャット一覧取得
// @Description 認証済みユーザーが作成したチャット一覧を取得します
// @Tags chats
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string][]model.Chat
// @Failure 401 {object} map[string]string "未認証"
// @Failure 403 {object} map[string]string "認可エラー・トークン不正"
// @Failure 500 {object} map[string]string "サーバーエラー"
// @Router /api/v1/chats [get]
func (h *ChatHandler) ListChats(c *gin.Context) {
	userID := auth.MustGetUserID(c)

	chats, err := h.chatService.ListChats(c.Request.Context(), userID)
	if err != nil {
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
// @Success 200 {object} model.Chat
// @Failure 400 {object} map[string]string "不正なID"
// @Failure 401 {object} map[string]string "未認証"
// @Failure 403 {object} map[string]string "認可エラー・トークン不正"
// @Failure 404 {object} map[string]string "チャットが見つからない"
// @Router /api/v1/chats/{id} [get]
func (h *ChatHandler) GetChat(c *gin.Context) {
	userID := auth.MustGetUserID(c)

	chatID, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chat id"})
		return
	}

	chat, err := h.chatService.GetChat(c.Request.Context(), chatID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "chat not found"})
		return
	}

	c.JSON(http.StatusOK, chat)
}

type SendMessageRequest struct {
	Content  string         `json:"content" binding:"required"`
	Provider model.Provider `json:"provider" binding:"required"` // "openrouter", "openai", "anthropic", "google"
	Model    string         `json:"model" binding:"required"`    // e.g. "anthropic/claude-3.5-sonnet"
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
// @Success 200 {object} map[string]model.Message
// @Failure 400 {object} map[string]string "不正なリクエスト"
// @Failure 401 {object} map[string]string "未認証"
// @Failure 403 {object} map[string]string "認可エラー・トークン不正"
// @Failure 500 {object} map[string]string "サーバーエラー"
// @Router /api/v1/chats/{id}/messages [post]
func (h *ChatHandler) SendMessage(c *gin.Context) {
	userID := auth.MustGetUserID(c)

	chatID, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chat id"})
		return
	}

	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content, provider, and model are required"})
		return
	}

	provider, err := model.ParseProvider(string(req.Provider))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Provider = provider

	userMsg, aiMsg, err := h.chatService.SendMessage(c.Request.Context(), chatID, userID, req.Content, req.Provider, req.Model)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
// @Failure 500 {object} map[string]string "サーバーエラー"
// @Router /api/v1/chats/{id}/messages/stream [post]
func (h *ChatHandler) StreamMessage(c *gin.Context) {
	userID := auth.MustGetUserID(c)

	chatID, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chat id"})
		return
	}

	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content, provider, and model are required"})
		return
	}

	provider, err := model.ParseProvider(string(req.Provider))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Provider = provider

	userMsg, textStream, onComplete, err := h.chatService.StreamMessage(c.Request.Context(), chatID, userID, req.Content, req.Provider, req.Model)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
	c.SSEvent("user_message", userMsg)
	c.Writer.Flush()

	// 2. チャンクをリアルタイムに送信
	var fullText strings.Builder
	for chunk := range textStream.TextStream() {
		fullText.WriteString(chunk)
		c.SSEvent("chunk", gin.H{"chunk": chunk})
		c.Writer.Flush()
	}

	// 3. ストリームエラーの確認
	if err := textStream.Err(); err != nil {
		c.SSEvent("error", gin.H{"error": err.Error()})
		c.Writer.Flush()
		return
	}

	// 4. DB に完成した全文を保存
	aiMsg, err := onComplete(fullText.String())
	if err != nil {
		c.SSEvent("error", gin.H{"error": "failed to persist assistant message: " + err.Error()})
		c.Writer.Flush()
		return
	}

	// 5. 完了シグナルを通知
	c.SSEvent("done", aiMsg)
	c.Writer.Flush()
}

func parseUintParam(c *gin.Context, paramName string) (uint, error) {
	val := c.Param(paramName)
	parsed, err := strconv.ParseUint(val, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint(parsed), nil
}
