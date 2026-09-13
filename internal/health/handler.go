package health

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Handler struct {
	db *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

// Get godoc
// @Summary ヘルスチェック (DB疎通確認)
// @Description サーバーおよびデータベースの稼働状態を確認します
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 503 {object} map[string]string "データベース接続異常"
// @Router /health [get]
func (h *Handler) Get(c *gin.Context) {
	sqlDB, err := h.db.DB()
	if err != nil {
		slog.Error("health check failed: failed to get sql.DB", "error", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":   "error",
			"database": "unavailable",
		})
		return
	}

	if err := sqlDB.PingContext(c.Request.Context()); err != nil {
		slog.Error("health check failed: db ping error", "error", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":   "error",
			"database": "unreachable",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "ok",
		"database": "connected",
	})
}
