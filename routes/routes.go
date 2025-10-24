package routes

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mseongj/weather-reminder/handlers"
)

func SetupRoutes() *mux.Router {
	router := mux.NewRouter()

	// API 라우트
	router.HandleFunc("/getTodayWeather", handlers.GetTodayWeather).Methods("GET")
	router.HandleFunc("/getFutureWeather", handlers.GetFutureWeather).Methods("GET")
	router.HandleFunc("/getTopNews", handlers.GetTopNews).Methods("GET")

	// 정적 파일 제공을 위한 핸들러 추가
	// PathPrefix를 사용하여 / 경로 아래의 모든 요청을 처리합니다.
	// 이 핸들러는 public 디렉토리의 파일을 제공합니다.
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./public/")))

	return router
}