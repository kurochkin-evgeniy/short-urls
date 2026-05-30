package handler_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"short-urls/internal/handler"
	"short-urls/internal/middleware"
	"short-urls/internal/repository"
	"short-urls/internal/service"
)

// ExampleHandleCreateShortUrRequest демонстрирует POST / с телом text/plain.
func ExampleHandleCreateShortUrRequest() {
	s := service.NewShortUrlService("http://localhost:8080", repository.NewMapKeyValueStorage())
	h := handler.HandleCreateShortUrRequest(s)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://practicum.yandex.ru/"))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()
	h(rec, req)

	fmt.Println(rec.Code)
	// Output: 201
}

// ExampleHandleCreateShortUrRequest_json демонстрирует POST /api/shorten с JSON-телом.
func ExampleHandleCreateShortUrRequest_json() {
	s := service.NewShortUrlService("http://localhost:8080", repository.NewMapKeyValueStorage())
	h := handler.HandleCreateShortUrRequest(s)

	body := `{"url":"https://practicum.yandex.ru/"}`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h(rec, req)

	fmt.Println(rec.Code)
	// Output: 201
}

// ExampleHandleRedirectRequest демонстрирует GET /{id} с редиректом на оригинальный URL.
func ExampleHandleRedirectRequest() {
	s := service.NewShortUrlService("http://localhost:8080", repository.NewMapKeyValueStorage())
	created, err := s.CreateShortUrl("https://practicum.yandex.ru/", "")
	if err != nil {
		panic(err)
	}
	shortID := strings.TrimPrefix(created.ShortURL, "http://localhost:8080/")

	h := handler.HandleRedirectRequest(s)
	req := httptest.NewRequest(http.MethodGet, "/"+shortID, nil)
	rec := httptest.NewRecorder()
	h(rec, req)

	fmt.Println(rec.Code)
	// Output: 307
}

// ExampleHandleCreateBatchShortUrRequest демонстрирует POST /api/shorten/batch.
func ExampleHandleCreateBatchShortUrRequest() {
	s := service.NewShortUrlService("http://localhost:8080", repository.NewMapKeyValueStorage())
	h := handler.HandleCreateBatchShortUrRequest(s)

	body := `[{"correlation_id":"1","original_url":"https://example.com/1"}]`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h(rec, req)

	fmt.Println(rec.Code)
	// Output: 201
}

// ExampleHandleGetUserURLsRequest демонстрирует GET /api/user/urls для аутентифицированного пользователя.
func ExampleHandleGetUserURLsRequest() {
	s := service.NewShortUrlService("http://localhost:8080", repository.NewMapKeyValueStorage())
	auth := middleware.AuthMiddleware("example-secret")
	createHandler := auth(http.HandlerFunc(handler.HandleCreateShortUrRequest(s)))
	getHandler := auth(http.HandlerFunc(handler.HandleGetUserURLsRequest(s)))

	createReq := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.com/user"))
	createReq.Header.Set("Content-Type", "text/plain")
	createRec := httptest.NewRecorder()
	createHandler.ServeHTTP(createRec, createReq)

	cookie := createRec.Result().Cookies()[0]
	getReq := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	getReq.AddCookie(cookie)
	getRec := httptest.NewRecorder()
	getHandler.ServeHTTP(getRec, getReq)

	fmt.Println(getRec.Code)
	// Output: 200
}

// ExampleHandleDeleteUserURLsRequest демонстрирует DELETE /api/user/urls.
func ExampleHandleDeleteUserURLsRequest() {
	s := service.NewShortUrlService("http://localhost:8080", repository.NewMapKeyValueStorage())
	auth := middleware.AuthMiddleware("example-secret")
	createHandler := auth(http.HandlerFunc(handler.HandleCreateShortUrRequest(s)))
	deleteHandler := auth(http.HandlerFunc(handler.HandleDeleteUserURLsRequest(s)))

	createReq := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.com/delete"))
	createReq.Header.Set("Content-Type", "text/plain")
	createRec := httptest.NewRecorder()
	createHandler.ServeHTTP(createRec, createReq)

	createBody, err := io.ReadAll(createRec.Result().Body)
	if err != nil {
		panic(err)
	}
	shortID := strings.TrimPrefix(strings.TrimSpace(string(createBody)), "http://localhost:8080/")

	cookie := createRec.Result().Cookies()[0]
	delReq := httptest.NewRequest(http.MethodDelete, "/api/user/urls", strings.NewReader(`["`+shortID+`"]`))
	delReq.Header.Set("Content-Type", "application/json")
	delReq.AddCookie(cookie)
	delRec := httptest.NewRecorder()
	deleteHandler.ServeHTTP(delRec, delReq)

	fmt.Println(delRec.Code)
	// Output: 202
}
