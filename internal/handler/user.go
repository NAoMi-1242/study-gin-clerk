package handler

import (
    "net/http"

    "github.com/gin-gonic/gin"

    "study-gin-clerk/internal/auth"
)

type UserHandler struct{}

func NewUserHandler() *UserHandler {
    return &UserHandler{}
}

func (h *UserHandler) GetMe(c *gin.Context) {
    userID := auth.MustGetUserID(c)

    c.JSON(http.StatusOK, gin.H{
        "user_id": userID,
    })
}