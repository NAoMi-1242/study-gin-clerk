package auth

import "github.com/gin-gonic/gin"

const userIDKey = "user_id"

func SetUserID(c *gin.Context, userID string) {
	c.Set(userIDKey, userID)
}

// GetUserID retrieves the authenticated Clerk user ID from the Gin context if present.
func GetUserID(c *gin.Context) (string, bool) {
	value, ok := c.Get(userIDKey)
	if !ok {
		return "", false
	}

	userID, ok := value.(string)
	if !ok || userID == "" {
		return "", false
	}

	return userID, true
}

// MustGetUserID retrieves the authenticated user ID from context, panicking if missing.
func MustGetUserID(c *gin.Context) string {
	userID, ok := GetUserID(c)
	if !ok {
		panic("authenticated user_id not found or invalid")
	}

	return userID
}
