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

var (
	weatherCache = &WeatherCache{}
	httpClient   = &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
		},
	}
)

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

	resp, err := httpClient.Get(apiUrl)
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

	// 미리 필요한 크기로 맵 초기화
	grouped := make(map[string]*models.WeatherItem, len(rawData)/5)
	var mu sync.Mutex // 맵 동시성 제어를 위한 뮤텍스

	// 데이터를 청크로 나누어 처리
	chunkSize := 100
	chunks := len(rawData) / chunkSize
	if len(rawData)%chunkSize != 0 {
		chunks++
	}

	var wg sync.WaitGroup
	for i := 0; i < chunks; i++ {
		wg.Add(1)
		start := i * chunkSize
		end := start + chunkSize
		if end > len(rawData) {
			end = len(rawData)
		}

		go func(items []models.WeatherItemToReturn) {
			defer wg.Done()
			for _, item := range items {
				key := item.Date + item.Time

				mu.Lock()
				if _, exists := grouped[key]; !exists {
					grouped[key] = &models.WeatherItem{
						Date: item.Date,
						Time: item.Time,
					}
				}
				mu.Unlock()

				switch item.Category {
				case "SKY":
					mu.Lock()
					grouped[key].Sky = parseCategory("SKY", item.Value)
					mu.Unlock()
				case "PTY":
					mu.Lock()
					grouped[key].Pty = parseCategory("PTY", item.Value)
					mu.Unlock()
				case "TMP":
					mu.Lock()
					grouped[key].Tmp = item.Value + "℃"
					mu.Unlock()
				case "POP":
					mu.Lock()
					grouped[key].Pop = item.Value + "%"
					mu.Unlock()
				case "REH":
					mu.Lock()
					grouped[key].Humidity = item.Value + "%"
					mu.Unlock()
				}
			}
		}(rawData[start:end])
	}

	wg.Wait()

	// 결과 슬라이스 미리 할당
	result := make([]models.WeatherItem, 0, len(grouped))
	for _, weather := range grouped {
		result = append(result, *weather)
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

// 다음 예보 발표 시간 계산 함수
func getNextForecastTime() time.Time {
	now := time.Now()
	forecastHours := []int{2, 5, 8, 11, 14, 17, 20, 23}
	
	// 현재 시간 이후의 다음 발표 시간 찾기
	for _, hour := range forecastHours {
		if now.Hour() < hour {
			return time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, time.Local)
		}
	}
	
	// 다음날 02시로 설정
	return time.Date(now.Year(), now.Month(), now.Day()+1, 2, 0, 0, 0, time.Local)
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
	weatherCache.ExpiresAt = getNextForecastTime()
}

func GetWeathers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	
	// 캐시에서 데이터 확인
	if cachedData, ok := getFromCache(); ok {
		log.Printf("캐시된 날씨 데이터 사용 (만료 시간: %v)", weatherCache.ExpiresAt)
		renderWeatherData(w, cachedData)
		
		// 다음 예보 발표 시간이 10분 이내로 남은 경우에만 백그라운드 갱신
		nextForecastTime := getNextForecastTime()
		if time.Until(nextForecastTime) < 10*time.Minute {
			go func() {
				// 실제 발표 시간 + 5분까지 대기 (데이터 갱신 시간 고려)
				time.Sleep(time.Until(nextForecastTime.Add(5 * time.Minute)))
				
				result, err := WeatherDataParse()
				if err != nil {
					log.Printf("백그라운드 캐시 갱신 실패: %v", err)
					return
				}
				setCache(result)
				log.Printf("백그라운드 캐시 갱신 완료 (만료 시간: %v)", weatherCache.ExpiresAt)
			}()
		}
		return
	}

	// 캐시에 없으면 새로운 데이터 가져오기
	result, err := WeatherDataParse()
	if err != nil {
		log.Printf("날씨 데이터 가져오기 실패: %v", err)
		http.Error(w, "날씨 데이터를 가져오는데 실패했습니다. 잠시 후 다시 시도해주세요.", http.StatusInternalServerError)
		return
	}

	// 데이터 캐시에 저장
	setCache(result)
	log.Printf("새로운 날씨 데이터 캐시 저장 (만료 시간: %v)", weatherCache.ExpiresAt)
	
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

