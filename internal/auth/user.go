package auth

import "github.com/gin-gonic/gin"

const userIDKey = "user_id"

func SetUserID(c *gin.Context, userID string) {
    c.Set(userIDKey, userID)
}

func MustGetUserID(c *gin.Context) string {
    value, ok := c.Get(userIDKey)
    if !ok {
        panic("authenticated user_id not found")
    }

    userID, ok := value.(string)
    if !ok || userID == "" {
        panic("authenticated user_id is invalid")
    }

    return userID
}