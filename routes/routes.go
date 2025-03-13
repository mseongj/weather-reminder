package routes

import (
	"github.com/gorilla/mux"
	"github.com/mseongj/weather-reminder/handlers"
)

func SetupRoutes() *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/todos", handlers.GetTodos).Methods("GET")
	router.HandleFunc("/todo/{id}", handlers.GetTodo).Methods("GET")
	router.HandleFunc("/todo", handlers.CreateTodo).Methods("POST")
	// router.HandleFunc("/todo/{id}", handlers.UpdateTodo).Methods("PUT") // 기존 UpdateTodo
	router.HandleFunc("/todo/{id}/toggle", handlers.ToggleTodo).Methods("PUT") // ToggleTodo 추가
	router.HandleFunc("/todo/{id}", handlers.DeleteTodo).Methods("DELETE")
	return router
}
