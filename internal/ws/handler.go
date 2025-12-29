package ws

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins in development
		// TODO: Restrict in production
		return true
	},
}

const (
	// Time to wait for a pong response
	pongWait = 60 * time.Second
	// Send pings at this interval (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10
	// Maximum message size
	maxMessageSize = 512
)

// Handler handles WebSocket upgrade requests
func Handler(r *ghttp.Request) {
	// Extract JWT token from query param or header
	token := r.GetQuery("token").String()
	if token == "" {
		authHeader := r.GetHeader("Authorization")
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		}
	}

	if token == "" {
		r.Response.WriteStatus(http.StatusUnauthorized)
		r.Response.WriteJson(g.Map{"message": "Token required"})
		return
	}

	// Validate JWT and extract userId
	userId, err := validateToken(token)
	if err != nil {
		g.Log().Warningf(r.Context(), "WebSocket: JWT validation failed: %v", err)
		r.Response.WriteStatus(http.StatusUnauthorized)
		r.Response.WriteJson(g.Map{"message": "Invalid token"})
		return
	}

	// Get the underlying http.ResponseWriter - GoFrame wraps it
	// We need to use the raw writer for WebSocket upgrade
	var rawWriter http.ResponseWriter = r.Response.RawWriter()

	// Upgrade to WebSocket using raw writer
	conn, err := upgrader.Upgrade(rawWriter, r.Request, nil)
	if err != nil {
		g.Log().Errorf(r.Context(), "WebSocket: upgrade failed: %v", err)
		return
	}

	// Register connection
	hub := GetHub()
	hub.Register(userId, conn)

	g.Log().Infof(r.Context(), "WebSocket: client connected, userId=%s", userId)

	// Handle connection in goroutine
	go handleConnection(userId, conn)
}

// handleConnection manages the WebSocket connection lifecycle
func handleConnection(userId string, conn *websocket.Conn) {
	hub := GetHub()
	defer func() {
		hub.Unregister(userId, conn)
		conn.Close()
		g.Log().Infof(context.TODO(), "WebSocket: client disconnected, userId=%s", userId)
	}()

	// Configure connection
	conn.SetReadLimit(maxMessageSize)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	// Start ping ticker
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	// Read loop (handles incoming messages and detects disconnection)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					g.Log().Warningf(context.TODO(), "WebSocket: unexpected close error: %v", err)
				}
				return
			}
			// Handle incoming messages (if any)
			g.Log().Debugf(context.TODO(), "WebSocket: received message from %s: %s", userId, string(message))
		}
	}()

	// Ping loop
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				g.Log().Warningf(context.TODO(), "WebSocket: ping failed: %v", err)
				return
			}
		}
	}
}

// validateToken validates JWT and returns userId
func validateToken(tokenStr string) (string, error) {
	secret := getJwtSecret()

	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return secret, nil
	})
	if err != nil {
		return "", err
	}

	if !token.Valid {
		return "", jwt.ErrSignatureInvalid
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", jwt.ErrTokenMalformed
	}

	// Check token type (must be access token)
	if tokenType, _ := claims["type"].(string); tokenType != "" && tokenType != "access" {
		return "", jwt.ErrTokenInvalidClaims
	}

	userId, ok := claims["userId"].(string)
	if !ok || userId == "" {
		return "", jwt.ErrTokenInvalidClaims
	}

	return userId, nil
}

// getJwtSecret returns the JWT secret
func getJwtSecret() []byte {
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		return []byte(secret)
	}
	v, _ := g.Cfg().Get(context.Background(), "jwt.secret")
	return v.Bytes()
}
