package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/mseongj/weather-reminder/models"
)

var loadEnvOnce sync.Once

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
// 기존 getWeatherData 함수에 정렬 추가
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

	var result []models.WeatherItemToReturn

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
			return "🌤"
		case "3":
			return "🌥"
		case "4":
			return "☁"
		default:
			return "알 수 없음"
		}
	case "PTY":
		switch value {
		case "0":
			return "없음"
		case "1":
			return "🌧"
		case "2":
			return "🌧(비/눈)"
		case "3":
			return "🌨"
		case "4":
			return "🌧(소나기)"
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
	result, _ := WeatherDataParse()

	// 🌟 데이터를 날짜(Date)와 시간(Time) 기준으로 정렬
	sort.Slice(result, func(i, j int) bool {
		if result[i].Date != result[j].Date {
			return result[i].Date < result[j].Date // 날짜(Date) 기준 오름차순
		}
		return result[i].Time < result[j].Time // 시간이 같으면 시간(Time) 기준 오름차순
	})

	for _, item := range result {
	    fmt.Fprintf(w, `
			<div class="weather">
            <p>날짜: %s</p>
            <p>시간: %s</p>
            <p class="sky-status">%s</p>
            <p class="precipitation-status">%s</p>
            <p>기온: %s</p>
            <p>강수확률: %s</p>
            <p>습도: %s</p>
        </div>
			`, item.Date, item.Time, item.Sky, item.Pty, item.Tmp, item.Pop, item.Humidity)
    }
}

