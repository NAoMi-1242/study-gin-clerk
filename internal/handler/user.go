package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"study-gin-clerk/internal/auth"
	"study-gin-clerk/internal/service"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// GetMe godoc
// @Summary 自分のプロファイル取得
// @Description 認証済みユーザーのプロファイル情報およびシステムプロンプトを取得します
// @Tags users
// @Security BearerAuth
// @Produce json
// @Success 200 {object} model.UserProfile
// @Failure 401 {object} map[string]string "未認証"
// @Failure 403 {object} map[string]string "認可エラー・トークン不正"
// @Failure 500 {object} map[string]string "サーバーエラー"
// @Router /api/v1/me [get]
func (h *UserHandler) GetMe(c *gin.Context) {
	userID := auth.MustGetUserID(c)

	profile, err := h.userService.GetProfile(c.Request.Context(), userID)
	if err != nil {
		slog.Error("failed to get profile", "error", err, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get profile",
		})
		return
	}

	c.JSON(http.StatusOK, profile)
}

type UpdateSystemPromptRequest struct {
	SystemPrompt string `json:"system_prompt" binding:"max=10000"`
}

// UpdateSystemPrompt godoc
// @Summary システムプロンプト更新
// @Description ユーザー共通のデフォルトシステムプロンプトを更新・保存します
// @Tags users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body UpdateSystemPromptRequest true "システムプロンプト設定 (最大10000文字)"
// @Success 200 {object} model.UserProfile
// @Failure 400 {object} map[string]string "不正なリクエスト"
// @Failure 401 {object} map[string]string "未認証"
// @Failure 500 {object} map[string]string "サーバーエラー"
// @Router /api/v1/me/system-prompt [put]
func (h *UserHandler) UpdateSystemPrompt(c *gin.Context) {
	userID := auth.MustGetUserID(c)

	var req UpdateSystemPromptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body (system_prompt max 10000 characters)"})
		return
	}

	profile, err := h.userService.UpdateSystemPrompt(c.Request.Context(), userID, req.SystemPrompt)
	if err != nil {
		slog.Error("failed to update system prompt", "error", err, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update system prompt"})
		return
	}

	c.JSON(http.StatusOK, profile)
}