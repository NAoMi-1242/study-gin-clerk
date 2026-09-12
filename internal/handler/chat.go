package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"study-gin-clerk/internal/auth"
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
// @Description 新しい会話セッションを作成します
// @Tags chats
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateChatRequest false "チャット設定"
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create chat"})
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
	Content string `json:"content" binding:"required"`
}

// SendMessage godoc
// @Summary メッセージ送信＆AI返答生成
// @Description ユーザーメッセージを送信し、AI返答を自動生成して保存します
// @Tags chats
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Chat ID"
// @Param request body SendMessageRequest true "メッセージ内容"
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "content is required"})
		return
	}

	userMsg, aiMsg, err := h.chatService.SendMessage(c.Request.Context(), chatID, userID, req.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_message":      userMsg,
		"assistant_message": aiMsg,
	})
}

func parseUintParam(c *gin.Context, paramName string) (uint, error) {
	val := c.Param(paramName)
	parsed, err := strconv.ParseUint(val, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint(parsed), nil
}

