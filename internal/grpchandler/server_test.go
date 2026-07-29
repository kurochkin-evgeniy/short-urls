package grpchandler_test

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"

	"short-urls/internal/auth"
	"short-urls/internal/facade"
	"short-urls/internal/grpchandler"
	"short-urls/internal/grpcmiddleware"
	"short-urls/internal/repository"
	"short-urls/internal/service"
	"short-urls/pkg/shortenerv1"
)

const bufSize = 1024 * 1024

func startTestGRPCServer(t *testing.T, secret string) (*grpc.ClientConn, func()) {
	t.Helper()

	listener := bufconn.Listen(bufSize)
	svc := service.NewShortUrlService("http://localhost:8080", repository.NewMapKeyValueStorage())
	f := facade.New(svc)
	server := grpc.NewServer(
		grpc.UnaryInterceptor(grpcmiddleware.AuthUnaryInterceptor(secret)),
	)
	shortenerv1.RegisterShortenerServiceServer(server, grpchandler.NewShortenerServer(f))

	go func() {
		if err := server.Serve(listener); err != nil {
			t.Logf("grpc server stopped: %v", err)
		}
	}()

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	cleanup := func() {
		conn.Close()
		server.Stop()
	}
	return conn, cleanup
}

func TestShortenURL(t *testing.T) {
	conn, cleanup := startTestGRPCServer(t, "test-secret")
	defer cleanup()

	client := shortenerv1.NewShortenerServiceClient(conn)
	resp, err := client.ShortenURL(context.Background(), &shortenerv1.URLShortenRequest{
		Url: "https://example.com/grpc",
	})
	require.NoError(t, err)
	assert.True(t, len(resp.GetResult()) > 0)
}

func TestExpandURL(t *testing.T) {
	conn, cleanup := startTestGRPCServer(t, "test-secret")
	defer cleanup()

	client := shortenerv1.NewShortenerServiceClient(conn)
	shortResp, err := client.ShortenURL(context.Background(), &shortenerv1.URLShortenRequest{
		Url: "https://example.com/expand",
	})
	require.NoError(t, err)

	shortID := shortResp.GetResult()[len("http://localhost:8080/"):]
	expandResp, err := client.ExpandURL(context.Background(), &shortenerv1.URLExpandRequest{Id: shortID})
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/expand", expandResp.GetResult())
}

func TestListUserURLsWithAuthorization(t *testing.T) {
	const secret = "test-secret"
	conn, cleanup := startTestGRPCServer(t, secret)
	defer cleanup()

	client := shortenerv1.NewShortenerServiceClient(conn)
	userID := auth.GenerateUserID()
	token := auth.SignUserToken(userID, secret)
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", token))

	_, err := client.ShortenURL(ctx, &shortenerv1.URLShortenRequest{Url: "https://example.com/user-grpc"})
	require.NoError(t, err)

	listResp, err := client.ListUserURLs(ctx, &emptypb.Empty{})
	require.NoError(t, err)
	require.Len(t, listResp.GetUrl(), 1)
	assert.Equal(t, "https://example.com/user-grpc", listResp.GetUrl()[0].GetOriginalUrl())
}

func TestListUserURLsUnauthorized(t *testing.T) {
	conn, cleanup := startTestGRPCServer(t, "test-secret")
	defer cleanup()

	client := shortenerv1.NewShortenerServiceClient(conn)
	macToken := auth.SignUserToken("", "test-secret")
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", macToken))

	_, err := client.ListUserURLs(ctx, &emptypb.Empty{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Unauthenticated")
}
