// Package grpcmiddleware предоставляет gRPC-interceptor'ы для аутентификации.
package grpcmiddleware

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"short-urls/internal/auth"
)

// AuthUnaryInterceptor извлекает авторизационные данные из metadata и сохраняет их в контексте.
func AuthUnaryInterceptor(secret string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		token := authorizationTokenFromContext(ctx)
		userID, authWasPresent, authHasNoID := auth.ResolveUserFromToken(token, secret)
		if !authWasPresent {
			userID = auth.GenerateUserID()
		}
		ctx = auth.ContextWithUserID(ctx, userID, authWasPresent, authHasNoID)
		return handler(ctx, req)
	}
}

func authorizationTokenFromContext(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	values := md.Get("authorization")
	if len(values) == 0 {
		return ""
	}
	token := strings.TrimSpace(values[0])
	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		return strings.TrimSpace(token[7:])
	}
	return token
}
