package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"study-gin-clerk/internal/auth"
	"study-gin-clerk/internal/service"
)

type AIModelHandler struct {
	aiModelService *service.AIModelService
}

func NewAIModelHandler(aiModelService *service.AIModelService) *AIModelHandler {
	return &AIModelHandler{aiModelService: aiModelService}
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
func (h *AIModelHandler) ListModels(c *gin.Context) {
	userID := auth.MustGetUserID(c)
	refresh := c.Query("refresh") == "true"

	models, activeProviders, err := h.aiModelService.GetAvailableModels(c.Request.Context(), userID, refresh)
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
