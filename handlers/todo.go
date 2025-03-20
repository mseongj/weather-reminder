package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

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

func getDate() string {
	now := time.Now()
	return now.Format("20060102")
}
// 날씨 데이터를 받아오는 함수 (결과를 변수에 담아 리턴)
func getWeatherData() ([]models.WeatherItemToReturn, error) {

	apiUrl := fmt.Sprintf(
			"http://apis.data.go.kr/1360000/VilageFcstInfoService_2.0/getVilageFcst?serviceKey=%s&pageNo=1&numOfRows=1000&dataType=JSON&base_date=%s&base_time=%s&nx=%d&ny=%d",
			getAPIKEY(), getDate(), "1400", 60, 127,
	)

	resp, err := http.Get(apiUrl)
	if err != nil {
			return nil, fmt.Errorf("HTTP 요청 실패: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
			return nil, fmt.Errorf("응답 본문 읽기 실패: %v", err)
	}

	var weatherResp models.WeatherResponse
	if err := json.Unmarshal(body, &weatherResp); err != nil {
			return nil, fmt.Errorf("JSON 파싱 실패: %v", err)
	}

	// 결과를 담을 슬라이스 생성
	var result []models.WeatherItemToReturn

	// 반복문으로 결과를 슬라이스에 추가
	for _, item := range weatherResp.Response.Body.Items.Item {
			result = append(result, models.WeatherItemToReturn{
					Date:     item.FcstDate,
					Time:     item.FcstTime,
					Category: item.Category,
					Value:    item.FcstValue,
			})
	}

	return result, nil
}
// 카테고리 코드값 변환 함수
func parseCategory(category, value string) string {
	switch category {
	case "SKY":
		switch value {
		case "1":
			return "맑음"
		case "3":
			return "구름많음"
		case "4":
			return "흐림"
		default:
			return "알 수 없음"
		}
	case "PTY":
		switch value {
		case "0":
			return "없음"
		case "1":
			return "비"
		case "2":
			return "비/눈"
		case "3":
			return "눈"
		case "4":
			return "소나기"
		default:
			return "알 수 없음"
		}
	default:
		return value
	}
}

func WeatherDataParse() ([]models.WeatherItem, error) {
	rawData, err := getWeatherData()
	// 에러처리
	if err != nil {
		fmt.Printf("getWeatherData()에서 error: %v", err)
		return nil, err
	}
	// 결과가 비어있으면 실패 처리
	if len(rawData) == 0 {
		fmt.Printf("getWeatherData()가 비어있음.")
		return nil, err
	}

	type tempWeather struct {
		Sky      string
		Pty      string
		Tmp      string
		Pop      string
    Humidity string
    }

	grouped := make(map[string]*tempWeather)

	for _, item := range rawData {
        key := item.Date + item.Time

        if _, exists := grouped[key]; !exists {
            grouped[key] = &tempWeather{}
        }

        switch item.Category {
        case "SKY":
            grouped[key].Sky = parseCategory("SKY", item.Value)
        case "PTY":
            grouped[key].Pty = parseCategory("PTY", item.Value)
        case "TMP":
            grouped[key].Tmp = item.Value + "℃"
        case "POP":
            grouped[key].Pop = item.Value + "%"
        case "REH":
            grouped[key].Humidity = item.Value + "%"
        }
    }

	var result []models.WeatherItem

	for dateTime, weather := range grouped {
        date := dateTime[:8]
        time := dateTime[8:]
        
        result = append(result, models.WeatherItem{
            Date:     date,
            Time:     time,
            Sky:      weather.Sky,
            Pty:      weather.Pty,
            Tmp:      weather.Tmp,
            Pop:      weather.Pop,
            Humidity: weather.Humidity,
        })
    }

	return result, nil
}

func GetWeathers(w http.ResponseWriter, r *http.Request){
	w.Header().Set("Content-Type", "text/html")
	data, err := WeatherDataParse()
	if err != nil {
	    log.Fatal(err)
  }

	for _, item := range data {
	    fmt.Fprintf(w, `
			<div class="weather">
            <p>날짜: %s</p>
            <p>시간: %s</p>
            <p>하늘 상태: %s</p>
            <p>강수형태: %s</p>
            <p>기온: %s</p>
            <p>강수확률: %s</p>
            <p>습도: %s</p>
        </div>
			`, item.Date, item.Time, item.Sky, item.Pty, item.Tmp, item.Pop, item.Humidity)
    }
}


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
