package main

import (
	"fmt"
	"net/http"

	"github.com/mseongj/weather-reminder/routes"
)

// CORS 미들웨어
func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowedOrigin := "*" // 실제 사용할 클라이언트 도메인으로 변경 필요

		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, hx-request, hx-trigger, hx-current-url, hx-target, hx-delete")
		// w.Header().Set("Vary", "Origin") // 캐싱 문제 방지
		// `true` 설정 시, Access-Control-Allow-Origin에 * 대신 특정 도메인을 명시해야 함.
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		// OPTIONS 요청 (Preflight) 바로 응답
		// OPTIONS 요청을 별도로 처리 (Preflight 요청 대응)
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, hx-request, hx-trigger, hx-current-url, hx-target, hx-delete")
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	router := routes.SetupRoutes()
	fmt.Println("Server is running on http://localhost:8080")
	http.ListenAndServe(":8080", enableCORS(router))
}