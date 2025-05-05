package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/mirjalilova/voice_transcribe/internal/controller/http/token"
	"golang.org/x/exp/slog"
)

const (
	key          = "vctr"
	unauthorized = "unauthorized"
)

func NewAuth(enforcer *casbin.Enforcer) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		allow, err := CheckPermission(ctx.FullPath(), ctx.Request, enforcer)

		if err != nil {
			slog.Error("Error checking permission: %v", err)
			if ve, ok := err.(*jwt.ValidationError); ok && ve.Errors == jwt.ValidationErrorExpired {
				RequireRefresh(ctx)
			} else {
				RequirePermission(ctx)
			}
			return
		}

		if !allow {
			RequirePermission(ctx)
			return
		}

		claims, err := ExtractToken(ctx.Request)
		if err != nil {
			slog.Error("Error extracting token: %v", err)
			InvalidToken(ctx)
			return
		}

		var id string
		var ok bool
		id, ok = claims["id"].(string)
		if !ok {
			slog.Warn("id not found in claims")
		}
		ctx.Set("id", id)
		ctx.Set("claims", claims)

		ctx.Next()
	}
}

func OptionalAuth() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		jwtToken := ctx.Request.Header.Get("Authorization")
		if jwtToken != "" {
			claims, err := ExtractToken(ctx.Request)
			if err != nil {
				if strings.Contains(err.Error(), "token expired") {
					slog.Warn("Token expired")
					ctx.AbortWithStatusJSON(401, gin.H{"error": "Token expired"})
					return
				}

				slog.Error("Invalid token: %v", err)
				ctx.AbortWithStatusJSON(401, gin.H{"error": "Invalid token"})
				return
			}

			slog.Info("Extracted optional claims: %v", claims)
			ctx.Set("claims", claims)
		}

		ctx.Next()
	}
}

func ExtractToken(r *http.Request) (jwt.MapClaims, error) {
	jwtToken := r.Header.Get("Authorization")
	if jwtToken == "" {
		return nil, fmt.Errorf("access token missing")
	}
	if strings.Contains(jwtToken, "Basic") {
		return nil, fmt.Errorf("invalid token format")
	}

	tokenString := strings.TrimSpace(strings.TrimPrefix(jwtToken, "Bearer "))

	claims, err := token.ExtractClaim(tokenString)
	if err != nil {
		slog.Error("Error while extracting claims: %v", err)
		return nil, err
	}

	return claims, nil
}

func GetRole(r *http.Request) (string, error) {
	claims, err := ExtractToken(r)
	if err != nil {
		return unauthorized, err
	}

	role, ok := claims["role"].(string)
	if !ok {
		return unauthorized, errors.New("role claim not found")
	}
	return role, nil
}

func CheckPermission(path string, r *http.Request, enforcer *casbin.Enforcer) (bool, error) {
	role, err := GetRole(r)
	if err != nil {
		slog.Error("Error getting role from token", err)
		return false, err
	}

	allowed, err := enforcer.Enforce(role, path, r.Method)
	if err != nil {
		slog.Error("Error during Casbin enforce", err)
		return false, err
	}

	return allowed, nil
}

func InvalidToken(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
		"error": "Invalid token!",
	})
}

func RequirePermission(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
		"error": "Permission denied",
	})
}

func RequireRefresh(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"error": "Access token expired",
	})
}
