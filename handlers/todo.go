package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/mseongj/weather-reminder/models"
)

var loadEnvOnce sync.Once

var todos = []models.Todo{
	{ID: 1, Title: "Learn Go", Completed: true},
	{ID: 2, Title: "Build REST API", Completed: false},
	{ID: 3, Title: "Build Frontend", Completed: false},
	{ID: 4, Title: "Build Fullstack App", Completed: false},
}

func getAPIKEY() string {
	// .env 파일을 한 번만 로드하도록 sync.Once 사용
	loadEnvOnce.Do(func() {
			if err := godotenv.Load(".env"); err != nil {
					log.Fatal("Error loading .env file")
			}
	})

	apiKey := os.Getenv("API_KEY")

	return apiKey
}

// func GetAPIKEY() string {
// 	// .env 파일 로드
// 	err := godotenv.Load(".env")
// 	if err != nil {
// 			log.Fatal("Error loading .env file")
// 	}

// 	// 환경 변수에서 API 키 가져오기
// 	apiKey := os.Getenv("API_KEY")

// 	// API 키 출력 (테스트용)
// 	// fmt.Println("API Key:", apiKey)
// 	return apiKey
// }

func GetTodos(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	for _, todo := range todos {
		completedClass := ""
		if todo.Completed {
			completedClass = "completed"
		}
        completedText := "완료되지않음"
        if todo.Completed {
            completedText = "완료됨"
        }
        // <button hx-put="">%s</button> 추가
        fmt.Fprintf(w, `
        <div class="todo %s" id="todo-%d">
            <span>%s</span>
            <button hx-put="http://127.0.0.1:8080/todo/%d/toggle" 
                    hx-target="#todo-%d" 
                    hx-swap="outerHTML">%s</button>
            <button hx-delete="http://127.0.0.1:8080/todo/%d" hx-target="#todo-%d">Delete</button>
        </div>
    `, completedClass, todo.ID, todo.Title, todo.ID, todo.ID, completedText, todo.ID, todo.ID)
	}
}

// Todo 상태 업데이트 핸들러
func ToggleTodo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid Todo ID", http.StatusBadRequest)
		return
	}

	// 해당 ID의 투두를 찾아 상태 토글
	for i := range todos {
		if todos[i].ID == id {
			todos[i].Completed = !todos[i].Completed

			// 업데이트된 투두 HTML 반환
			w.Header().Set("Content-Type", "text/html")
			completedClass := ""
			if todos[i].Completed {
				completedClass = "completed"
			}
			completedText := "완료되지않음"
			if todos[i].Completed {
				completedText = "완료됨"
			}
			fmt.Fprintf(w, `
				<div class="todo %s" id="todo-%d">
					<span>%s</span>
					<button hx-put="http://127.0.0.1:8080/todo/%d/toggle" 
							hx-target="#todo-%d"
							hx-swap="outerHTML">%s</button>
					<button hx-delete="http://127.0.0.1:8080/todo/%d" hx-target="#todo-%d">Delete</button>
				</div>
			`, completedClass, todos[i].ID, todos[i].Title, todos[i].ID, todos[i].ID, completedText, todos[i].ID, todos[i].ID)
			return
		}
	}

	http.NotFound(w, r) // ID에 해당하는 Todo가 없을 경우
}

// Todo 삭제 핸들러
func DeleteTodo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid Todo ID", http.StatusBadRequest)
		return
	}

	// 해당 ID의 투두 삭제
	for i := range todos {
		if todos[i].ID == id {
			todos = append(todos[:i], todos[i+1:]...)
			w.WriteHeader(http.StatusOK) // 성공 응답
			return
		}
	}

	http.NotFound(w, r) // ID에 해당하는 Todo가 없을 경우
}


func GetTodo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	for _, todo := range todos {
		if todo.ID == id {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(todo)
			return
		}
	}

	http.Error(w, "Todo not found", http.StatusNotFound)
}

func CreateTodo(w http.ResponseWriter, r *http.Request) {
    if err := r.ParseForm(); err != nil {
        http.Error(w, "Failed to parse form data", http.StatusBadRequest)
        return
    }

    // HTMX는 기본적으로 form-urlencoded 방식으로 데이터를 보냄
    title := r.FormValue("title")
    if title == "" {
        http.Error(w, "Title is required", http.StatusBadRequest)
        return
    }

    // 새로운 Todo 생성
    todo := models.Todo{
        ID:    len(todos) + 1,
        Title: title,
    }
    todos = append(todos, todo)

    completedText := "완료되지않음"
    if todo.Completed {
        completedText = "완료됨"
    }

    // 반환할 HTML 조각 생성
    html := fmt.Sprintf(`
    <div class="todo" id="todo-%d">
        <span>%s</span>
        <span>%s</span>
        <button hx-delete="http://127.0.0.1:8080/todo/%d" hx-target="#todo-%d">Delete</button>
    </div>`, todo.ID, todo.Title, completedText, todo.ID, todo.ID)

    w.Header().Set("Content-Type", "text/html")
    w.WriteHeader(http.StatusCreated)
    w.Write([]byte(html))
}

func UpdateTodo(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    idStr := vars["id"]

    id, err := strconv.Atoi(idStr)
    if err != nil {
        http.Error(w, "Invalid ID", http.StatusBadRequest)
        return
    }

    for i, todo := range todos {
        if todo.ID == id {
            todos[i].Completed = !todos[i].Completed // 상태 토글

            // 완료 여부 텍스트 설정
            completedText := "완료되지않음"
            if todos[i].Completed {
                completedText = "완료됨"
            }

            // 업데이트된 HTML 반환
            w.Header().Set("Content-Type", "text/html")
            fmt.Fprintf(w, `
                <div class="todo" id="todo-%d">
                    <span>%s</span>
                    <span>%s</span>
                    <button hx-delete="http://127.0.0.1:8080/todo/%d" hx-target="#todo-%d">Delete</button>
                </div>
            `, todos[i].ID, todos[i].Title, completedText, todos[i].ID, todos[i].ID)
            return
        }
    }

    http.Error(w, "Todo not found", http.StatusNotFound)
}
