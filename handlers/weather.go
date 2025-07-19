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

type RequestMetrics struct {
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
}

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
	loadEnvOnce.Do(func() {
		if err := godotenv.Load(".env"); err != nil {
			log.Fatal("Error loading .env file")
		}
	})
	return os.Getenv("API_KEY")
}

func getBaseDateTime() (string, string) {
	now := time.Now().Add(-10 * time.Minute)
	hour := now.Hour()
	baseDate := now.Format("20060102")
	var baseTime string
	switch {
	case hour < 2:
		yesterday := now.AddDate(0, 0, -1)
		baseDate = yesterday.Format("20060102")
		baseTime = "2300"
	case hour < 5:
		baseTime = "0200"
	case hour < 8:
		baseTime = "0500"
	case hour < 11:
		baseTime = "0800"
	case hour < 14:
		baseTime = "1100"
	case hour < 17:
		baseTime = "1400"
	case hour < 20:
		baseTime = "1700"
	case hour < 23:
		baseTime = "2000"
	default:
		baseTime = "2300"
	}
	return baseDate, baseTime
}

func getWeatherData() ([]models.WeatherItemToReturn, error) {
	metrics := RequestMetrics{StartTime: time.Now()}
	baseDate, baseTime := getBaseDateTime()
	apiUrl := fmt.Sprintf(
		"http://apis.data.go.kr/1360000/VilageFcstInfoService_2.0/getVilageFcst?serviceKey=%s&pageNo=1&numOfRows=900&dataType=JSON&base_date=%s&base_time=%s&nx=%d&ny=%d",
		getAPIKEY(), baseDate, baseTime, 77, 131,
	)

	resp, err := httpClient.Get(apiUrl)
	if err != nil {
		return nil, fmt.Errorf("HTTP 요청 실패: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API 응답 실패: 상태 코드 %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("응답 본문 읽기 실패: %v", err)
	}

	var weatherResp models.WeatherResponse
	if err := json.Unmarshal(body, &weatherResp); err != nil {
		log.Printf("JSON 파싱 실패. 응답 내용: %s", string(body))
		return nil, fmt.Errorf("JSON 파싱 실패: %v", err)
	}

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

	metrics.EndTime = time.Now()
	metrics.Duration = metrics.EndTime.Sub(metrics.StartTime)
	log.Printf("날씨 데이터 요청 처리 시간: %v", metrics.Duration)
	return result, nil
}

func parseCategory(category, value string) string {
	switch category {
	case "SKY":
		switch value {
		case "1": return "🌤"
		case "3": return "🌥"
		case "4": return "☁"
		default: return "알 수 없음"
		}
	case "PTY":
		switch value {
		case "0": return "none"
		case "1": return "🌧"
		case "2": return "🌧(비/눈)"
		case "3": return "🌨"
		case "4": return "🌧(소나기)"
		default: return "알 수 없음"
		}
	default:
		return value
	}
}

func WeatherDataParse() ([]models.WeatherItem, error) {
	rawData, err := getWeatherData()
	if err != nil {
		return nil, fmt.Errorf("getWeatherData()에서 error: %v", err)
	}
	if len(rawData) == 0 {
		return nil, fmt.Errorf("getWeatherData()가 비어있음")
	}

	grouped := make(map[string]*models.WeatherItem, len(rawData)/5)
	for _, item := range rawData {
		key := item.Date + item.Time
		if _, exists := grouped[key]; !exists {
			grouped[key] = &models.WeatherItem{Date: item.Date, Time: item.Time}
		}
		switch item.Category {
		case "SKY": grouped[key].Sky = parseCategory("SKY", item.Value)
		case "PTY": grouped[key].Pty = parseCategory("PTY", item.Value)
		case "TMP": grouped[key].Tmp = item.Value + "℃"
		case "POP": grouped[key].Pop = item.Value + "%"
		case "REH": grouped[key].Humidity = item.Value + "%"
		}
	}

	result := make([]models.WeatherItem, 0, len(grouped))
	for _, weather := range grouped {
		result = append(result, *weather)
	}
	return result, nil
}

func formatTime(timeStr string) string {
	return fmt.Sprintf("%s시", timeStr[:2])
}

func getTempClass(tempStr string) string {
	tempStr = strings.TrimSuffix(tempStr, "℃")
	temp, err := strconv.Atoi(tempStr)
	if err != nil {
		return "temp-cold"
	}
	if temp <= 10 { return "temp-cold" }
	if temp <= 20 { return "temp-cool" }
	if temp <= 30 { return "temp-warm" }
	return "temp-hot"
}

func getNextForecastTime() time.Time {
	now := time.Now()
	forecastHours := []int{2, 5, 8, 11, 14, 17, 20, 23}
	for _, hour := range forecastHours {
		if now.Hour() < hour {
			return time.Date(now.Year(), now.Month(), now.Day(), hour, 10, 0, 0, time.Local) // 10분 마진
		}
	}
	return time.Date(now.Year(), now.Month(), now.Day()+1, 2, 10, 0, 0, time.Local)
}

func getFromCache() ([]models.WeatherItem, bool) {
	weatherCache.mutex.RLock()
	defer weatherCache.mutex.RUnlock()
	if time.Now().Before(weatherCache.ExpiresAt) {
		return weatherCache.Data, true
	}
	return nil, false
}

func setCache(data []models.WeatherItem) {
	weatherCache.mutex.Lock()
	defer weatherCache.mutex.Unlock()
	weatherCache.Data = data
	weatherCache.ExpiresAt = getNextForecastTime()
}

func fetchAndCacheWeather() ([]models.WeatherItem, error) {
    if cachedData, ok := getFromCache(); ok {
        log.Printf("캐시된 날씨 데이터 사용 (만료 시간: %v)", weatherCache.ExpiresAt)
        return cachedData, nil
    }

    result, err := WeatherDataParse()
    if err != nil {
        log.Printf("날씨 데이터 가져오기 실패: %v", err)
        return nil, err
    }

    setCache(result)
    log.Printf("새로운 날씨 데이터 캐시 저장 (만료 시간: %v)", weatherCache.ExpiresAt)
    return result, nil
}

func GetTodayWeather(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/html")
    allWeather, err := fetchAndCacheWeather()
    if err != nil {
        http.Error(w, "날씨 정보를 가져올 수 없습니다.", http.StatusInternalServerError)
        return
    }

    today := time.Now().Format("20060102")
    var todayWeather []models.WeatherItem
    for _, item := range allWeather {
        if item.Date == today {
            todayWeather = append(todayWeather, item)
        }
    }
    
    sort.Slice(todayWeather, func(i, j int) bool {
        return todayWeather[i].Time < todayWeather[j].Time
    })

    renderTodayWeather(w, todayWeather)
}

func GetFutureWeather(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/html")
    allWeather, err := fetchAndCacheWeather()
    if err != nil {
        http.Error(w, "날씨 정보를 가져올 수 없습니다.", http.StatusInternalServerError)
        return
    }

    today := time.Now().Format("20060102")
    groupedByDate := make(map[string][]models.WeatherItem)
    for _, item := range allWeather {
        if item.Date != today {
            groupedByDate[item.Date] = append(groupedByDate[item.Date], item)
        }
    }

    var sortedDates []string
    for date := range groupedByDate {
        sortedDates = append(sortedDates, date)
    }
    sort.Strings(sortedDates)

    maxDays := 2
    if len(sortedDates) > maxDays {
        sortedDates = sortedDates[:maxDays]
    }

    renderFutureWeather(w, sortedDates, groupedByDate)
}

func renderTodayWeather(w http.ResponseWriter, items []models.WeatherItem) {
	fmt.Fprint(w, `<div class="weather-grid">`)
	for _, item := range items {
		displayIcon := item.Sky
		if item.Pty != "none" {
			displayIcon = item.Pty
		}
		tempClass := getTempClass(item.Tmp)
		fmt.Fprintf(w, `
            <div class="weather">
                <p class="sky-status">%s</p>
                <p class="temp %s">%s</p>
                <p class="rain-chance">강수확률: %s</p>
                <p class="humidity">습도: %s</p>
                <p class="time">%s</p>
            </div>`,
			displayIcon, tempClass, item.Tmp, item.Pop, item.Humidity, formatTime(item.Time))
	}
	fmt.Fprint(w, `</div>`)
}

func renderFutureWeather(w http.ResponseWriter, dates []string, data map[string][]models.WeatherItem) {
    for _, date := range dates {
        items := data[date]
        sort.Slice(items, func(i, j int) bool {
            return items[i].Time < items[j].Time
        })

        formattedDate := fmt.Sprintf("%s월 %s일", date[4:6], date[6:8])
        fmt.Fprintf(w, `<div class="date-group">
            <h3 class="date-title">%s</h3>
            <div class="weather-grid">`, formattedDate)

        for _, item := range items {
            timeInt, _ := strconv.Atoi(item.Time[:2])
            if timeInt % 2 == 0 { // 2시간 간격으로 표시
                displayIcon := item.Sky
				if item.Pty != "none" {
					displayIcon = item.Pty
				}
                tempClass := getTempClass(item.Tmp)
                fmt.Fprintf(w, `
                    <div class="weather">
                        <p class="sky-status">%s</p>
                        <p class="temp %s">%s</p>
                        <p class="rain-chance">강수: %s</p>
                        <p class="time">%s</p>
                    </div>`,
                    displayIcon, tempClass, item.Tmp, item.Pop, formatTime(item.Time))
            }
        }
        fmt.Fprint(w, `</div></div>`)
    }
}