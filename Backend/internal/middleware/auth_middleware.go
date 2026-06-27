package middleware

import (
	"net/http"
	"strings"

	"bank-service/internal/config"
	"bank-service/internal/database"
	jwtProvider "bank-service/internal/infrastructure/jwt"

	"github.com/gin-gonic/gin"
)

type userSession struct {
	SessionVersion int
}

// AuthMiddleware kiểm tra access token từ Authorization header
func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "Thiếu Authorization header",
			})
			c.Abort()
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "Authorization header không hợp lệ",
			})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		claims, err := jwtProvider.ValidateToken(
			tokenString,
			cfg.AccessTokenSecret,
		)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "Access token không hợp lệ hoặc đã hết hạn",
			})
			c.Abort()
			return
		}

		var user userSession

		err = database.DB.
			Table("users").
			Select("session_version").
			Where("id = ?", claims.UserID).
			First(&user).Error

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "Không thể xác thực phiên đăng nhập",
			})
			c.Abort()
			return
		}

		if claims.SessionVersion != user.SessionVersion {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "Phiên đăng nhập đã hết hiệu lực",
			})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)
		c.Set("session_version", claims.SessionVersion)

		c.Next()
	}
}
