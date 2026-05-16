package main

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
	"github.com/mark3labs/mcp-go/server"
	"schedule-mcp/db"
)

// EchoSessionMiddleware extracts session_id from cookie and hydrates context
func EchoSessionMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		cookie, err := c.Cookie("session_id")
		if err != nil {
			if strings.HasPrefix(c.Request().URL.Path, "/api/") {
				return c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: "Unauthorized"})
			}
			return next(c)
		}

		// Parse session ID into pgtype.UUID
		var sessionID pgtype.UUID
		if err := parseUUID(cookie.Value, &sessionID); err != nil {
			if strings.HasPrefix(c.Request().URL.Path, "/api/") {
				return c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: "Invalid session"})
			}
			return next(c)
		}

		u, err := queries.GetUserBySessionID(c.Request().Context(), db.GetUserBySessionIDParams{
			ID:        sessionID,
			ExpiresAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
		})

		if err != nil {
			if strings.HasPrefix(c.Request().URL.Path, "/api/") {
				return c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: "Unauthorized"})
			}
			return next(c)
		}

		user := &User{
			ID:        u.ID,
			Email:     u.Email.String,
			APIKey:    u.ApiKey,
			Role:      u.Role.String,
			Tier:      u.Tier.String,
			CreatedAt: u.CreatedAt.Time,
		}

		log.Printf("Session Validated: UserID=%s, Email=%s, Role=%s", user.ID, user.Email, user.Role)

		c.Set("user", user)
		c.Set("user_id", user.ID)
		c.Set("user_role", user.Role)

		// Also add to request context for downstream non-echo handlers if any
		ctx := context.WithValue(c.Request().Context(), userKey, user)
		ctx = context.WithValue(ctx, userIDKey, user.ID)
		ctx = context.WithValue(ctx, userRoleKey, user.Role)
		c.SetRequest(c.Request().WithContext(ctx))

		return next(c)
	}
}

// EchoRequireRole ensures the user has one of the required roles
func EchoRequireRole(roles ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userRole, ok := c.Get("user_role").(string)
			if !ok || userRole == "" {
				log.Printf("RBAC Denial: No user_role found in context for %s", c.Request().URL.Path)
				if strings.HasPrefix(c.Request().URL.Path, "/api/") {
					return c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: "Unauthorized"})
				}
				return echo.NewHTTPError(http.StatusForbidden, "Forbidden")
			}

			for _, role := range roles {
				if userRole == role {
					return next(c)
				}
			}

			log.Printf("RBAC Denial: User role '%s' not in allowed list %v for %s", userRole, roles, c.Request().URL.Path)
			if strings.HasPrefix(c.Request().URL.Path, "/api/") {
				return c.JSON(http.StatusForbidden, APIResponse{Success: false, Error: "Forbidden - Insufficient permissions"})
			}
			return echo.NewHTTPError(http.StatusForbidden, "Forbidden")
		}
	}
}

// EchoAuthMiddleware ensures every request has a valid X-API-Key linked to a user
func EchoAuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		apiKey := c.Request().Header.Get("X-API-Key")
		if apiKey == "" {
			return echo.NewHTTPError(http.StatusUnauthorized, "Unauthorized: Missing X-API-Key")
		}

		u, err := queries.GetUserByAPIKey(c.Request().Context(), apiKey)
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "Unauthorized: Invalid API Key")
		}

		// Phase 4: Rate Limiting
		if !globalRateLimiter.Allow(c.Request().Context(), u.ID) {
			return echo.NewHTTPError(http.StatusTooManyRequests, "Too Many Requests")
		}

		// Add UserID and Tier to context for use in tools
		c.Set("user_id", u.ID)
		c.Set("user_tier", u.Tier.String)

		ctx := context.WithValue(c.Request().Context(), userIDKey, u.ID)
		ctx = context.WithValue(ctx, userTierKey, u.Tier.String)
		c.SetRequest(c.Request().WithContext(ctx))

		return next(c)
	}
}

// EchoRateLimitMiddleware applies the global rate limiter based on the user ID in context
func EchoRateLimitMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		userID := getUserID(c)
		if userID != "" {
			if !globalRateLimiter.Allow(c.Request().Context(), userID) {
				return c.JSON(http.StatusTooManyRequests, APIResponse{Success: false, Error: "Too Many Requests"})
			}
		}
		return next(c)
	}
}

func getUserFromEcho(c echo.Context) *User {
	user, _ := c.Get("user").(*User)
	return user
}

func getUserID(c echo.Context) string {
	if user := getUserFromEcho(c); user != nil {
		return user.ID
	}
	id, _ := c.Get("user_id").(string)
	return id
}

// NetHttpAuthMiddleware is a wrapper to use authentication logic for standard library handlers (SSE/Message).
// It supports both X-API-Key header and session_id cookie.
func NetHttpAuthMiddleware(next http.Handler, mcpServer *server.MCPServer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := ""
		userTier := ""

		// 1. Try API Key Header
		apiKey := r.Header.Get("X-API-Key")
		if apiKey != "" {
			u, err := queries.GetUserByAPIKey(r.Context(), apiKey)
			if err == nil {
				userID = u.ID
				userTier = u.Tier.String
			}
		}

		// 2. Try Session Cookie if API key failed
		if userID == "" {
			cookie, err := r.Cookie("session_id")
			if err == nil && cookie.Value != "" {
				var sessionID pgtype.UUID
				if err := parseUUID(cookie.Value, &sessionID); err == nil {
					u, err := queries.GetUserBySessionID(r.Context(), db.GetUserBySessionIDParams{
						ID:        sessionID,
						ExpiresAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
					})
					if err == nil {
						userID = u.ID
						userTier = u.Tier.String
					}
				}
			}
		}

		if userID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if !globalRateLimiter.Allow(r.Context(), userID) {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		ctx = context.WithValue(ctx, userTierKey, userTier)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
