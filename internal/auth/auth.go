// Package auth предоставляет подпись токенов пользователя и хранение идентификатора в контексте.
package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"time"
)

type contextKey string

const (
	userIDContextKey   contextKey = "userID"
	authSeenContextKey contextKey = "authSeen"
	authNoIDContextKey contextKey = "authNoID"
	defaultSecret                 = "short-urls-secret"
)

// ContextWithUserID возвращает контекст с идентификатором пользователя и флагами авторизации.
func ContextWithUserID(ctx context.Context, userID string, authWasPresent bool, authHasNoID bool) context.Context {
	ctx = context.WithValue(ctx, userIDContextKey, userID)
	ctx = context.WithValue(ctx, authSeenContextKey, authWasPresent)
	ctx = context.WithValue(ctx, authNoIDContextKey, authHasNoID)
	return ctx
}

// UserIDFromContext возвращает идентификатор пользователя из контекста.
func UserIDFromContext(ctx context.Context) string {
	value := ctx.Value(userIDContextKey)
	if value == nil {
		return ""
	}
	userID, ok := value.(string)
	if !ok {
		return ""
	}
	return userID
}

// AuthWasPresent сообщает, были ли переданы авторизационные данные.
func AuthWasPresent(ctx context.Context) bool {
	value := ctx.Value(authSeenContextKey)
	if value == nil {
		return false
	}
	wasPresent, ok := value.(bool)
	if !ok {
		return false
	}
	return wasPresent
}

// AuthHasNoID сообщает, что авторизационные данные были переданы, но не содержали идентификатор пользователя.
func AuthHasNoID(ctx context.Context) bool {
	value := ctx.Value(authNoIDContextKey)
	if value == nil {
		return false
	}
	hasNoID, ok := value.(bool)
	if !ok {
		return false
	}
	return hasNoID
}

// ResolveUserFromToken извлекает идентификатор пользователя из подписанного токена.
func ResolveUserFromToken(token string, secret string) (userID string, authWasPresent bool, authHasNoID bool) {
	if token == "" {
		return "", false, false
	}

	verifiedID, ok, hasID := VerifySignedUserToken(token, secret)
	if !ok {
		return GenerateUserID(), true, false
	}
	if !hasID {
		return "", true, true
	}
	return verifiedID, true, false
}

// SignUserToken подписывает идентификатор пользователя.
func SignUserToken(userID string, secret string) string {
	return signUserToken(userID, normalizeSecret(secret))
}

// VerifySignedUserToken проверяет подпись токена и возвращает идентификатор пользователя.
func VerifySignedUserToken(token string, secret string) (string, bool, bool) {
	return verifySignedUserToken(token, normalizeSecret(secret))
}

// GenerateUserID создаёт новый случайный идентификатор пользователя.
func GenerateUserID() string {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return hex.EncodeToString([]byte(time.Now().String()))
	}
	return hex.EncodeToString(buffer)
}

func normalizeSecret(secret string) string {
	if secret == "" {
		return defaultSecret
	}
	return secret
}

func signUserToken(userID string, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(userID))
	signature := mac.Sum(nil)
	raw := userID + "." + hex.EncodeToString(signature)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func verifySignedUserToken(token string, secret string) (string, bool, bool) {
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", false, false
	}

	parts := strings.SplitN(string(decoded), ".", 2)
	if len(parts) != 2 {
		return "", false, false
	}

	userID := parts[0]
	signatureHex := parts[1]

	signature, err := hex.DecodeString(signatureHex)
	if err != nil {
		return "", false, false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(userID))
	expected := mac.Sum(nil)
	if !hmac.Equal(signature, expected) {
		return "", false, false
	}

	return userID, true, userID != ""
}
