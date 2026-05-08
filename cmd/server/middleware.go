package main

import (
	"context"
	"net/http"

	"github.com/mark3labs/mcp-go/server"
)

// authMiddleware ensures every request has a valid X-API-Key linked to a user
func authMiddleware(next http.Handler, mcpServer *server.MCPServer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			http.Error(w, "Unauthorized: Missing X-API-Key", http.StatusUnauthorized)
			return
		}

		var userID, userTier string
		err := dbPool.QueryRow(r.Context(), "SELECT id, tier FROM users WHERE api_key = $1", apiKey).Scan(&userID, &userTier)
		if err != nil {
			http.Error(w, "Unauthorized: Invalid API Key", http.StatusUnauthorized)
			return
		}

		// Phase 4: Rate Limiting
		if !globalRateLimiter.Allow(r.Context(), userID) {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		// Phase 6.2: Distributed Session Tracking via Redis
		// MaintainHeartbeat will block until r.Context() is cancelled (client disconnects)
		go GlobalSessionManager.MaintainHeartbeat(r.Context(), userID, mcpServer)

		// Add UserID and Tier to context for use in tools
		ctx := context.WithValue(r.Context(), "user_id", userID)
		ctx = context.WithValue(ctx, "user_tier", userTier)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
