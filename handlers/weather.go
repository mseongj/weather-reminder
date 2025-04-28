package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/mseongj/weather-reminder/models"
)

var loadEnvOnce sync.Once

// 요청 처리 시간을 측정하기 위한 구조체
type RequestMetrics struct {
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
}

// 캐시 구조체 정의
type WeatherCache struct {
	Data      []models.WeatherItem
	ExpiresAt time.Time
	mutex     sync.RWMutex
}

// 전역 캐시 변수
var weatherCache = &WeatherCache{}

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
	// 요청 시작 시간 기록
	metrics := RequestMetrics{
		StartTime: time.Now(),
	}

	apiUrl := fmt.Sprintf(
		"http://apis.data.go.kr/1360000/VilageFcstInfoService_2.0/getVilageFcst?serviceKey=%s&pageNo=1&numOfRows=900&dataType=JSON&base_date=%s&base_time=%s&nx=%d&ny=%d",
		getAPIKEY(), getDate(), "1400", 77, 131, // 강원 홍천 화촌면 (77, 131)
		// 대구 도원동 (88, 89)
	)

	resp, err := http.Get(apiUrl)
	if err != nil {
		return nil, fmt.Errorf("HTTP 요청 실패: %v", err)
	}
	defer resp.Body.Close()

	// 응답 상태 코드 확인
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API 응답 실패: 상태 코드 %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("응답 본문 읽기 실패: %v", err)
	}

	var weatherResp models.WeatherResponse
	if err := json.Unmarshal(body, &weatherResp); err != nil {
		// JSON 파싱 실패 시 응답 내용을 로그에 기록
		log.Printf("JSON 파싱 실패. 응답 내용: %s", string(body))
		return nil, fmt.Errorf("JSON 파싱 실패: %v", err)
	}

	// 응답이 비어있는지 확인
	if len(weatherResp.Response.Body.Items.Item) == 0 {
		return nil, fmt.Errorf("API 응답이 비어있습니다")
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

	// 요청 종료 시간 기록 및 로깅
	metrics.EndTime = time.Now()
	metrics.Duration = metrics.EndTime.Sub(metrics.StartTime)
	log.Printf("날씨 데이터 요청 처리 시간: %v", metrics.Duration)

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
			return "none"
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

// formatTime 함수 추가
func formatTime(timeStr string) string {
	hour := timeStr[:2]
	return fmt.Sprintf("%s시", hour)
}

func getTempClass(tempStr string) string {
	// 온도 문자열에서 숫자만 추출
	tempStr = strings.TrimSuffix(tempStr, "℃")
	temp, err := strconv.Atoi(tempStr)
	if err != nil {
		return "temp-cold"
	}

	// 온도에 따른 클래스 반환
	if temp <= 10 {
		return "temp-cold"
	} else if temp <= 20 {
		return "temp-cool"
	} else if temp <= 30 {
		return "temp-warm"
	} else {
		return "temp-hot"
	}
}

// 캐시 만료 시간 계산 함수
func calculateExpiryTime() time.Time {
	now := time.Now()
	// 다음 예보 시간 계산 (3시간 단위)
	nextHour := ((now.Hour() / 3) + 1) * 3
	if nextHour >= 24 {
		nextHour = 0
		now = now.AddDate(0, 0, 1)
	}
	return time.Date(now.Year(), now.Month(), now.Day(), nextHour, 0, 0, 0, time.Local)
}

// 캐시에서 데이터 가져오기
func getFromCache() ([]models.WeatherItem, bool) {
	weatherCache.mutex.RLock()
	defer weatherCache.mutex.RUnlock()

	if time.Now().Before(weatherCache.ExpiresAt) {
		return weatherCache.Data, true
	}
	return nil, false
}

// 캐시에 데이터 저장
func setCache(data []models.WeatherItem) {
	weatherCache.mutex.Lock()
	defer weatherCache.mutex.Unlock()

	weatherCache.Data = data
	weatherCache.ExpiresAt = calculateExpiryTime()
}

func GetWeathers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	
	// 캐시에서 데이터 확인
	if cachedData, ok := getFromCache(); ok {
		renderWeatherData(w, cachedData)
		return
	}

	// 캐시에 없으면 새로운 데이터 가져오기
	result, err := WeatherDataParse()
	if err != nil {
		http.Error(w, "날씨 데이터를 가져오는데 실패했습니다", http.StatusInternalServerError)
		return
	}

	// 데이터 캐시에 저장
	setCache(result)
	
	renderWeatherData(w, result)
}

// 날씨 데이터 렌더링 함수 분리
func renderWeatherData(w http.ResponseWriter, result []models.WeatherItem) {
	sort.Slice(result, func(i, j int) bool {
		if result[i].Date != result[j].Date {
			return result[i].Date < result[j].Date
		}
		return result[i].Time < result[j].Time
	})

	groupedByDate := make(map[string][]models.WeatherItem)
	for _, item := range result {
		groupedByDate[item.Date] = append(groupedByDate[item.Date], item)
	}
	
	var sortedDates []string
	for date := range groupedByDate {
		sortedDates = append(sortedDates, date)
	}
	sort.Strings(sortedDates)

	// 최대 3일까지만 표시
	maxDays := 3
	if len(sortedDates) > maxDays {
		sortedDates = sortedDates[:maxDays]
	}

	for i, date := range sortedDates {
		items := groupedByDate[date]
		formattedDate := fmt.Sprintf("%s년 %s월 %s일", 
			date[:4], 
			date[4:6], 
			date[6:8])
		
		// 첫 번째 날짜(오늘)는 날짜 제목을 표시하지 않음
		if i == 0 {
			fmt.Fprintf(w, `<div class="date-group">
				<div class="weather-grid">`)
		} else {
			fmt.Fprintf(w, `<div class="date-group">
				<h3 class="date-title">%s</h3>
				<div class="weather-grid">`, formattedDate)
		}
		
		for _, item := range items {
			// 첫 번째 날짜(오늘)는 모든 시간을 표시
			// 그 외 날짜는 짝수 시간만 표시
			timeInt, _ := strconv.Atoi(item.Time[:2])
			if i == 0 || timeInt%2 == 0 {
				var 강수형태 string
				if item.Pty == "none" {
					강수형태 = ""
				} else {
					강수형태 = fmt.Sprintf("<p class='precipitation-status'>%s</p>", item.Pty)
				}
				
				tempClass := getTempClass(item.Tmp)
				
				fmt.Fprintf(w, `
					<div class="weather">
						<p class="sky-status">%s</p>
						%s
						<p class="temp %s">%s</p>
						<p class="rain-chance">강수확률: %s</p>
						<p class="humidity">습도: %s</p>
						<p class="time">%s</p>
					</div>`,
					item.Sky, 강수형태, tempClass, item.Tmp, item.Pop, item.Humidity, formatTime(item.Time))
			}
		}
		fmt.Fprintf(w, `</div></div>`)
	}
}

