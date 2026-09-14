package auth

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAuthContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("GetUserID when present", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		SetUserID(c, "user_12345")

		id, ok := GetUserID(c)
		if !ok || id != "user_12345" {
			t.Errorf("got (%q, %v), want (user_12345, true)", id, ok)
		}

		mustID := MustGetUserID(c)
		if mustID != "user_12345" {
			t.Errorf("got %q, want user_12345", mustID)
		}
	})

	t.Run("GetUserID when not set", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		id, ok := GetUserID(c)
		if ok || id != "" {
			t.Errorf("expected empty string and false, got (%q, %v)", id, ok)
		}
	})

	t.Run("MustGetUserID panics when not set", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		defer func() {
			r := recover()
			if r == nil {
				t.Errorf("expected panic for missing user ID, but did not panic")
			}
		}()

		_ = MustGetUserID(c)
	})
}

