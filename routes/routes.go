package routes

import (
	"github.com/gorilla/mux"
	"github.com/mseongj/weather-reminder/handlers"
)

func SetupRoutes() *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/getweathers", handlers.GetWeathers).Methods("GET")
	return router
}
