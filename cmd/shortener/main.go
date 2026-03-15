package main

import (
	"io"
	"math/rand"
	"net/http"
)

type Service struct {
	dict map[string]string
}

var s Service

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func randStringBytesSafe(m map[string]string) string {
	for {
		s := randStringBytes(6)

		if _, ok := m[s]; !ok {
			return s
		}
	}
}

func handleCreateShortUrl(responce http.ResponseWriter, request *http.Request) {

	ct := request.Header.Get("content-type")
	if ct == "text/plain" && request.URL.Path == "/" {
		body, err := io.ReadAll(request.Body)
		if err == nil {
			id := randStringBytesSafe(s.dict)

			s.dict[id] = string(body)
			responce.WriteHeader(http.StatusCreated)
			responce.Write([]byte("http://localhost:8080/" + id))
			return
		}
	}

	responce.WriteHeader(http.StatusBadRequest)
}

func handleRedirectUrl(responce http.ResponseWriter, request *http.Request) {

	id := request.PathValue("id")
	url := s.dict[id]
	if url != "" {
		responce.Header().Add("Location", url)
		responce.WriteHeader(http.StatusTemporaryRedirect)
		return
	}

	responce.WriteHeader(http.StatusBadRequest)
}

func routeUrl(responce http.ResponseWriter, request *http.Request) {
	switch request.Method {
	case http.MethodPost:
		handleCreateShortUrl(responce, request)
	case http.MethodGet:
		handleRedirectUrl(responce, request)
	default:
		responce.WriteHeader(http.StatusBadRequest)
	}
}

func main() {

	s.dict = make(map[string]string)

	mux := http.NewServeMux()
	mux.HandleFunc("/{id}", routeUrl)
	mux.HandleFunc("/", routeUrl)

	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		panic(err)
	}
}
