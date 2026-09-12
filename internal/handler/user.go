package handler

import (
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
// @Description 認証済みユーザーのプロファイル情報を取得します
// @Tags users
// @Security BearerAuth
// @Produce json
// @Success 200 {object} service.UserProfile
// @Failure 401 {object} map[string]string "未認証"
// @Failure 403 {object} map[string]string "認可エラー・トークン不正"
// @Router /api/v1/me [get]
func (h *UserHandler) GetMe(c *gin.Context) {
	userID := auth.MustGetUserID(c)

	profile, err := h.userService.GetProfile(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get profile",
		})
		return
	}

	c.JSON(http.StatusOK, profile)
}