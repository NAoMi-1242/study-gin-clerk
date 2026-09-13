package handler

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"study-gin-clerk/internal/auth"
	"study-gin-clerk/internal/model"
	"study-gin-clerk/internal/service"
)

type UserAPIKeyHandler struct {
	keyService *service.UserAPIKeyService
}

func NewUserAPIKeyHandler(keyService *service.UserAPIKeyService) *UserAPIKeyHandler {
	return &UserAPIKeyHandler{keyService: keyService}
}

type RegisterAPIKeyRequest struct {
	Provider model.Provider `json:"provider" binding:"required"` // "openrouter", "openai", "anthropic", "google"
	APIKey   string         `json:"api_key" binding:"required,max=500"`
}

// RegisterKey godoc
// @Summary API キー登録・更新
// @Description ユーザー独自の各社 AI API キーを登録します。登録時にプロバイダへの疎通確認を行い、AES-256-GCM で暗号化して安全に保存します。
// @Tags api_keys
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body RegisterAPIKeyRequest true "API キー情報"
// @Success 201 {object} map[string]string "登録成功 (マスクされた key_hint を返却)"
// @Failure 400 {object} map[string]string "検証失敗または不正なリクエスト"
// @Failure 401 {object} map[string]string "未認証"
// @Failure 500 {object} map[string]string "サーバーエラー"
// @Router /api/v1/user/api-keys [post]
func (h *UserAPIKeyHandler) RegisterKey(c *gin.Context) {
	userID := auth.MustGetUserID(c)

	var req RegisterAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider and api_key (max 500 chars) are required"})
		return
	}

	provider, err := model.ParseProvider(string(req.Provider))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Provider = provider

	record, err := h.keyService.RegisterKey(c.Request.Context(), userID, req.Provider, req.APIKey)
	if err != nil {
		if strings.Contains(err.Error(), "validation failed") || strings.Contains(err.Error(), "unsupported provider") || strings.Contains(err.Error(), "required") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.Error("failed to register API key", "error", err, "user_id", userID, "provider", req.Provider)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save API key"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"provider":   record.Provider,
		"key_hint":   record.KeyHint,
		"updated_at": record.UpdatedAt,
	})
}

// ListKeys godoc
// @Summary 登録済み API キー一覧
// @Description ユーザーが登録した各社 API キーの一覧（マスクされた key_hint のみ）を取得します。
// @Tags api_keys
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string][]model.UserAPIKey
// @Failure 401 {object} map[string]string "未認証"
// @Failure 500 {object} map[string]string "サーバーエラー"
// @Router /api/v1/user/api-keys [get]
func (h *UserAPIKeyHandler) ListKeys(c *gin.Context) {
	userID := auth.MustGetUserID(c)

	keys, err := h.keyService.ListKeys(c.Request.Context(), userID)
	if err != nil {
		slog.Error("failed to list API keys", "error", err, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list API keys"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"keys": keys})
}

// DeleteKey godoc
// @Summary 登録済み API キー削除
// @Description 指定されたプロバイダの API キーを削除し、モデルキャッシュをパージします。
// @Tags api_keys
// @Security BearerAuth
// @Produce json
// @Param provider path string true "Provider Name (e.g. openrouter)"
// @Success 200 {object} map[string]string "削除成功"
// @Failure 400 {object} map[string]string "不正なリクエスト"
// @Failure 401 {object} map[string]string "未認証"
// @Failure 404 {object} map[string]string "キーが見つからない"
// @Failure 500 {object} map[string]string "サーバーエラー"
// @Router /api/v1/user/api-keys/{provider} [delete]
func (h *UserAPIKeyHandler) DeleteKey(c *gin.Context) {
	userID := auth.MustGetUserID(c)
	provider, err := model.ParseProvider(c.Param("provider"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.keyService.DeleteKey(c.Request.Context(), userID, provider); err != nil {
		slog.Warn("API key deletion failed or not found", "error", err, "user_id", userID, "provider", provider)
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found for provider: " + string(provider)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "API key deleted successfully"})
}
