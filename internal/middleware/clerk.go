package middleware

import (
	"net/http"

	"github.com/clerk/clerk-sdk-go/v2"
	clerkhttp "github.com/clerk/clerk-sdk-go/v2/http"
	"github.com/gin-gonic/gin"

	"study-gin-clerk/internal/auth"
)

func ClerkAuthMiddleware() gin.HandlerFunc {
	clerkMiddleware := clerkhttp.RequireHeaderAuthorization()

	return func(c *gin.Context) {
		called := false
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			// Clerk middlewareが追加したContextを
			// GinのRequestにも引き継ぐ。
			c.Request = r

			claims, ok := clerk.SessionClaimsFromContext(r.Context())

			// 認証済みでuser_idを取得できることを
			// Handlerに渡す前に保証する。
			if !ok || claims == nil || claims.Subject == "" {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "authentication required",
				})
				return
			}

			// Clerk固有の情報からuser_idを取得し、
			// アプリケーション側の認証情報として保存する。
			auth.SetUserID(c, claims.Subject)

			c.Next()
		})

		clerkMiddleware(next).ServeHTTP(c.Writer, c.Request)
		if !called {
			c.Abort()
		}
	}
}
