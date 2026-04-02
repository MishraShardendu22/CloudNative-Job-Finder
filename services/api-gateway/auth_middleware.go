package main

import (
	"context"
	"net/http"
	"strings"

	"job-finder/shared/auth"
	"job-finder/shared/httpx"
)

func (a *app) jwtMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublicRoute(r.Method, r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			httpx.WriteError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}
		token := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := auth.ParseToken(a.jwtSecret, token)
		if err != nil {
			httpx.WriteError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		ctx := context.WithValue(r.Context(), claimsContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func isPublicRoute(method, path string) bool {
	if method == http.MethodGet && path == "/health" {
		return true
	}
	if method == http.MethodPost && (path == "/signup" || path == "/login") {
		return true
	}
	return false
}

func getClaims(ctx context.Context) *auth.Claims {
	claims, _ := ctx.Value(claimsContextKey).(*auth.Claims)
	return claims
}
