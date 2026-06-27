package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const userIDKey contextKey = "user_id"
const userRoleKey contextKey = "user_role"
const requestIDKey contextKey = "request_id"

func UserID(ctx context.Context) int64 {
	v, _ := ctx.Value(userIDKey).(int64)
	return v
}

func UserRole(ctx context.Context) string {
	v, _ := ctx.Value(userRoleKey).(string)
	return v
}

func RequestID(ctx context.Context) string {
	v, _ := ctx.Value(requestIDKey).(string)
	return v
}

// Claims is the typed JWT claims structure.
type Claims struct {
	jwt.RegisteredClaims
	Role string `json:"role"`
}

func (h *Handler) RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := r.Header.Get("X-Request-ID")
		if rid == "" {
			rid = generateRequestID()
		}
		w.Header().Set("X-Request-ID", rid)
		ctx := context.WithValue(r.Context(), requestIDKey, rid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *Handler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			writeError(w, http.StatusUnauthorized, "missing token")
			return
		}
		tokenStr := strings.TrimPrefix(auth, "Bearer ")

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return h.jwtSecret, nil
		})
		if err != nil || !token.Valid {
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		uid, _ := claims.GetSubject()
		userID, err := strconv.ParseInt(uid, 10, 64)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid token subject")
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		ctx = context.WithValue(ctx, userRoleKey, claims.Role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *Handler) RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !allowed[UserRole(r.Context())] {
				writeError(w, http.StatusForbidden, "insufficient role")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
