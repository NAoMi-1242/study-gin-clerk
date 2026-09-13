package apikey

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"study-gin-clerk/internal/auth"
	"study-gin-clerk/internal/types"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type RegisterAPIKeyRequest struct {
	Provider types.Provider `json:"provider" binding:"required"` // "openrouter", "openai", "anthropic", "google"
	APIKey   string         `json:"api_key" binding:"required,max=500"`
}

// RegisterKey godoc
// @Summary API キー登録・更新
// @Description ユーザー独自の各社 AI API キーを登録します。登録時にプロバイダへの疎通確認を行い、AES-256-GCM で暗号化して安全に保存します。
// @Tags user_api_keys
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body RegisterAPIKeyRequest true "API キー情報"
// @Success 201 {object} map[string]string "登録成功 (マスクされた key_hint を返却)"
// @Failure 400 {object} map[string]string "検証失敗または不正なリクエスト"
// @Failure 401 {object} map[string]string "未認証"
// @Failure 500 {object} map[string]string "サーバーエラー"
// @Router /api/v1/me/api-keys [post]
func (h *Handler) RegisterKey(c *gin.Context) {
	userID := auth.MustGetUserID(c)

	var req RegisterAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider and api_key (max 500 chars) are required"})
		return
	}

	provider, err := types.ParseProvider(string(req.Provider))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Provider = provider
	req.APIKey = strings.TrimSpace(req.APIKey)
	if req.APIKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "api_key cannot be empty or whitespace only"})
		return
	}

	record, err := h.service.RegisterKey(c.Request.Context(), userID, req.Provider, req.APIKey)
	if err != nil {
		if errors.Is(err, ErrValidationFailed) {
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
// @Tags user_api_keys
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string][]Key
// @Failure 401 {object} map[string]string "未認証"
// @Failure 500 {object} map[string]string "サーバーエラー"
// @Router /api/v1/me/api-keys [get]
func (h *Handler) ListKeys(c *gin.Context) {
	userID := auth.MustGetUserID(c)

	keys, err := h.service.ListKeys(c.Request.Context(), userID)
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
// @Tags user_api_keys
// @Security BearerAuth
// @Produce json
// @Param provider path string true "Provider Name (e.g. openrouter)"
// @Success 200 {object} map[string]string "削除成功"
// @Failure 400 {object} map[string]string "不正なリクエスト"
// @Failure 401 {object} map[string]string "未認証"
// @Failure 404 {object} map[string]string "キーが見つからない"
// @Failure 500 {object} map[string]string "サーバーエラー"
// @Router /api/v1/me/api-keys/{provider} [delete]
func (h *Handler) DeleteKey(c *gin.Context) {
	userID := auth.MustGetUserID(c)
	provider, err := types.ParseProvider(c.Param("provider"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.DeleteKey(c.Request.Context(), userID, provider); err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "API key not found for provider: " + string(provider)})
			return
		}
		if errors.Is(err, ErrValidationFailed) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.Error("failed to delete API key", "error", err, "user_id", userID, "provider", provider)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete API key"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "API key deleted successfully"})
}
