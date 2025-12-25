package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/bison/api-server/pkg/logger"
)

// AuthHandler handles authentication
type AuthHandler struct {
	username  string
	password  string
	jwtSecret []byte
	enabled   bool
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(username, password, jwtSecret string, enabled bool) *AuthHandler {
	return &AuthHandler{
		username:  username,
		password:  password,
		jwtSecret: []byte(jwtSecret),
		enabled:   enabled,
	}
}

// LoginRequest represents login request
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents login response
type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expiresAt"`
	Username  string `json:"username"`
}

// Login handles user login
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("Login failed: invalid request", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名和密码不能为空", "code": "INVALID_REQUEST"})
		return
	}

	// Validate credentials
	if req.Username != h.username || req.Password != h.password {
		logger.Warn("Login failed: invalid credentials", "username", req.Username)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误", "code": "INVALID_CREDENTIALS"})
		return
	}

	// Generate JWT token
	expiresAt := time.Now().Add(24 * time.Hour)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": req.Username,
		"exp":      expiresAt.Unix(),
		"iat":      time.Now().Unix(),
	})

	tokenString, err := token.SignedString(h.jwtSecret)
	if err != nil {
		logger.Error("Login failed: token generation error", "error", err, "username", req.Username)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成令牌失败", "code": "TOKEN_GENERATION_FAILED"})
		return
	}

	logger.Info("User logged in", "username", req.Username)
	c.JSON(http.StatusOK, LoginResponse{
		Token:     tokenString,
		ExpiresAt: expiresAt.Unix(),
		Username:  req.Username,
	})
}

// GetAuthStatus returns the current auth status
func (h *AuthHandler) GetAuthStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"authEnabled": h.enabled,
	})
}

// AuthMiddleware returns a JWT authentication middleware
func (h *AuthHandler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// If auth is disabled, allow all requests
		if !h.enabled {
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未提供认证令牌", "code": "NO_TOKEN"})
			c.Abort()
			return
		}

		// Parse Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "认证令牌格式错误", "code": "INVALID_TOKEN_FORMAT"})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Parse and validate JWT token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return h.jwtSecret, nil
		})

		if err != nil || !token.Valid {
			logger.Debug("Auth failed: invalid token", "error", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "认证令牌无效或已过期", "code": "INVALID_TOKEN"})
			c.Abort()
			return
		}

		// Extract claims
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			c.Set("username", claims["username"])
		}

		c.Next()
	}
}
