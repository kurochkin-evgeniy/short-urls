// Package grpchandler реализует gRPC-обработчики сервиса сокращения URL.
package grpchandler

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"short-urls/internal/facade"
	"short-urls/internal/logging"
	"short-urls/pkg/shortenerv1"
)

// ShortenerServer реализует gRPC-сервис ShortenerService.
type ShortenerServer struct {
	shortenerv1.UnimplementedShortenerServiceServer
	facade *facade.Shortener
}

// NewShortenerServer создаёт gRPC-сервер поверх общего фасада.
func NewShortenerServer(f *facade.Shortener) *ShortenerServer {
	return &ShortenerServer{facade: f}
}

// ShortenURL обрабатывает запрос на сокращение URL.
func (s *ShortenerServer) ShortenURL(ctx context.Context, req *shortenerv1.URLShortenRequest) (*shortenerv1.URLShortenResponse, error) {
	result, _, err := s.facade.ShortenURL(ctx, req.GetUrl())
	if err != nil {
		logging.Sugar.Errorw("Failed to create short url", "error", err)
		return nil, status.Error(codes.Internal, "failed to create short url")
	}
	return &shortenerv1.URLShortenResponse{Result: result}, nil
}

// ExpandURL возвращает оригинальный URL по идентификатору короткой ссылки.
func (s *ShortenerServer) ExpandURL(ctx context.Context, req *shortenerv1.URLExpandRequest) (*shortenerv1.URLExpandResponse, error) {
	result, err := s.facade.ExpandURL(ctx, req.GetId())
	if err != nil {
		switch {
		case errors.Is(err, facade.ErrNotFound):
			return nil, status.Error(codes.InvalidArgument, "short url not found")
		case errors.Is(err, facade.ErrGone):
			return nil, status.Error(codes.FailedPrecondition, "short url deleted")
		default:
			logging.Sugar.Errorw("Failed to expand short url", "error", err)
			return nil, status.Error(codes.Internal, "failed to expand short url")
		}
	}
	return &shortenerv1.URLExpandResponse{Result: result}, nil
}

// ListUserURLs возвращает все активные короткие ссылки пользователя.
func (s *ShortenerServer) ListUserURLs(ctx context.Context, _ *emptypb.Empty) (*shortenerv1.UserURLsResponse, error) {
	userURLs, err := s.facade.ListUserURLs(ctx)
	if err != nil {
		if errors.Is(err, facade.ErrUnauthorized) {
			return nil, status.Error(codes.Unauthenticated, "unauthorized")
		}
		logging.Sugar.Errorw("Failed to read user urls", "error", err)
		return nil, status.Error(codes.Internal, "failed to read user urls")
	}

	response := &shortenerv1.UserURLsResponse{
		Url: make([]*shortenerv1.URLData, 0, len(userURLs)),
	}
	for _, item := range userURLs {
		response.Url = append(response.Url, &shortenerv1.URLData{
			ShortUrl:    item.ShortURL,
			OriginalUrl: item.OriginalURL,
		})
	}
	return response, nil
}
