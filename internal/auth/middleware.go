package auth

import (
	"net/http"

	"github.com/clerk/clerk-sdk-go/v2"
	clerkhttp "github.com/clerk/clerk-sdk-go/v2/http"
	"github.com/gin-gonic/gin"
)

// RequireAuth returns a Gin middleware that validates the Clerk session JWT
// from the Authorization header and stores the authenticated user ID in the context.
func RequireAuth() gin.HandlerFunc {
	clerkMiddleware := clerkhttp.RequireHeaderAuthorization()

	return func(c *gin.Context) {
		called := false
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			c.Request = r

			claims, ok := clerk.SessionClaimsFromContext(r.Context())
			if !ok || claims == nil || claims.Subject == "" {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "authentication required",
				})
				return
			}

			SetUserID(c, claims.Subject)
			c.Next()
		})

		clerkMiddleware(next).ServeHTTP(c.Writer, c.Request)
		if !called {
			c.Abort()
		}
	}
}

