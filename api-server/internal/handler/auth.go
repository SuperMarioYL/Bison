package handler

import (
	"crypto/subtle"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/bison/api-server/pkg/logger"
)

// Login brute-force protection: after maxLoginFails failed attempts from one IP
// within loginWindow, that IP is locked out for loginBlock.
const (
	maxLoginFails = 5
	loginWindow   = 5 * time.Minute
	loginBlock    = 15 * time.Minute
)

type failRecord struct {
	count        int
	resetAt      time.Time
	blockedUntil time.Time
}

// loginLimiter is a small in-memory per-IP failed-login limiter.
type loginLimiter struct {
	mu    sync.Mutex
	fails map[string]*failRecord
}

func newLoginLimiter() *loginLimiter {
	return &loginLimiter{fails: make(map[string]*failRecord)}
}

// allowed reports whether the IP may attempt a login now; if blocked it returns
// the number of seconds to wait.
func (l *loginLimiter) allowed(ip string, now time.Time) (bool, int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	rec := l.fails[ip]
	if rec != nil && now.Before(rec.blockedUntil) {
		return false, int(rec.blockedUntil.Sub(now).Seconds()) + 1
	}
	return true, 0
}

// recordFailure increments the failure counter for an IP and blocks it once the
// threshold within the window is exceeded.
func (l *loginLimiter) recordFailure(ip string, now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	rec := l.fails[ip]
	if rec == nil || now.After(rec.resetAt) {
		rec = &failRecord{resetAt: now.Add(loginWindow)}
		l.fails[ip] = rec
	}
	rec.count++
	if rec.count >= maxLoginFails {
		rec.blockedUntil = now.Add(loginBlock)
	}
	// Opportunistic prune to bound memory.
	if len(l.fails) > 1024 {
		for k, v := range l.fails {
			if now.After(v.resetAt) && now.After(v.blockedUntil) {
				delete(l.fails, k)
			}
		}
	}
}

func (l *loginLimiter) recordSuccess(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.fails, ip)
}

// AuthHandler handles authentication
type AuthHandler struct {
	username  string
	password  string
	jwtSecret []byte
	enabled   bool
	limiter   *loginLimiter
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(username, password, jwtSecret string, enabled bool) *AuthHandler {
	return &AuthHandler{
		username:  username,
		password:  password,
		jwtSecret: []byte(jwtSecret),
		enabled:   enabled,
		limiter:   newLoginLimiter(),
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
	ip := c.ClientIP()
	now := time.Now()

	// Reject brute-force attempts before doing any credential work.
	if ok, retryAfter := h.limiter.allowed(ip, now); !ok {
		logger.Warn("Login blocked: too many failed attempts", "ip", ip, "retryAfterSec", retryAfter)
		c.Header("Retry-After", strconv.Itoa(retryAfter))
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "登录尝试过于频繁，请稍后再试", "code": "TOO_MANY_ATTEMPTS"})
		return
	}

	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("Login failed: invalid request", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名和密码不能为空", "code": "INVALID_REQUEST"})
		return
	}

	// Validate credentials using constant-time comparison to avoid leaking timing
	// information. Both comparisons always run so username validity is not revealed.
	userOK := subtle.ConstantTimeCompare([]byte(req.Username), []byte(h.username)) == 1
	passOK := subtle.ConstantTimeCompare([]byte(req.Password), []byte(h.password)) == 1
	if !userOK || !passOK {
		h.limiter.recordFailure(ip, now)
		logger.Warn("Login failed: invalid credentials", "username", req.Username, "ip", ip)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误", "code": "INVALID_CREDENTIALS"})
		return
	}
	h.limiter.recordSuccess(ip)

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
