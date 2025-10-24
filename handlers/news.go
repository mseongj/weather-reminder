package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp" // 정규표현식(Regex) 패키지 추가
	"strings"
	"sync"
	"time"

	"github.com/mseongj/weather-reminder/models"
)

type NewsCache struct {
    Data      []models.NewsItem // NewsItem 슬라이스를 캐싱
    ExpiresAt time.Time
    mutex     sync.RWMutex
}

var (
    newsCache = &NewsCache{}
)

// 중복 제거를 위해 뉴스 제목의 [속보], (종합) 등을 제거하는 정규표현식
var newsTitleCleaner = regexp.MustCompile(`^\[.*?\]|\(.*?\)`)

// Naver API 응답에 포함된 HTML 태그(<b>, &lt; 등)를 제거하는 헬퍼 함수
func cleanHtmlTags(s string) string {
	s = strings.ReplaceAll(s, "<b>", "")
	s = strings.ReplaceAll(s, "</b>", "")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	return s
}

// 기사 목록에서 중복을 제거하고 원하는 개수만큼 반환하는 함수
func filterUniqueArticles(items []models.NewsItem, maxItems int) []models.NewsItem {
	// 최종 반환될 고유 기사 슬라이스
	uniqueArticles := make([]models.NewsItem, 0, maxItems)
	
	// 중복 체크를 위한 맵 (Key: 정리된 기사 제목)
	addedTitles := make(map[string]bool)

	for _, item := range items {
		// 1. <b> 태그 등 HTML 먼저 제거
		cleanedTitle := cleanHtmlTags(item.Title)
		
		// 2. [속보] (종합) 같은 머릿말 제거
		cleanedTitle = newsTitleCleaner.ReplaceAllString(cleanedTitle, "")
		
		// 3. 앞뒤 공백 제거
		cleanedTitle = strings.TrimSpace(cleanedTitle)

		// 4. 이미 맵에 존재하는 제목인지 확인
		if _, exists := addedTitles[cleanedTitle]; exists {
			continue // 이미 추가된 기사(중복)이므로 건너뜀
		}

		// 5. 새로운 기사인 경우
		addedTitles[cleanedTitle] = true // 맵에 등록
		uniqueArticles = append(uniqueArticles, item) // 결과 슬라이스에 추가

		// 6. 원하는 개수(5개)를 채웠으면 반복 중단
		if len(uniqueArticles) >= maxItems {
			break
		}
	}
	return uniqueArticles
}

// --- ⭐️ 2. 신규 함수: 실제 API 호출 로직 분리 ---
// (기존 GetTopNews의 API 호출 부분을 이 함수로 이동)
func fetchNewsFromAPI() ([]models.NewsItem, error) {
	// 1. Naver API 인증 정보 가져오기
	clientID := os.Getenv("NAVER_CLIENT_ID")
	clientSecret := os.Getenv("NAVER_CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("Naver API ID 또는 Secret이 설정되지 않았습니다")
	}

	// 2. 검색어 설정 및 API URL 준비
	query := "뉴스"
	escapedQuery := url.QueryEscape(query)
	apiURL := fmt.Sprintf("https://openapi.naver.com/v1/search/news.json?query=%s&display=20&sort=sim", escapedQuery)

	// 3. HTTP 요청 생성 (GET)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("naver API 요청 생성 실패: %v", err)
	}

	// 4. HTTP 헤더에 Client ID와 Secret 추가
	req.Header.Add("X-Naver-Client-Id", clientID)
	req.Header.Add("X-Naver-Client-Secret", clientSecret)

	// 5. 요청 보내기 (weather.go의 공유 httpClient 사용)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Naver API 요청 실패: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Naver API 응답 코드: %d", resp.StatusCode)
	}

	// 6. 응답 본문 읽기 및 JSON 파싱
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Naver API 응답 읽기 실패: %v", err)
	}

	var newsResp models.NaverNewsResponse
	if err := json.Unmarshal(body, &newsResp); err != nil {
		return nil, fmt.Errorf("Naver JSON 파싱 실패: %v", err)
	}

	// 7. API 응답(20개)에서 중복을 제거하고 5개만 필터링
	uniqueArticles := filterUniqueArticles(newsResp.Items, 5)
	
	return uniqueArticles, nil
}

func getNewsFromCache() ([]models.NewsItem, bool) {
	newsCache.mutex.RLock()
	defer newsCache.mutex.RUnlock()
	if time.Now().Before(newsCache.ExpiresAt) {
		log.Println("캐시된 뉴스 데이터 사용") // 확인용 로그
		return newsCache.Data, true
	}
	return nil, false
}

func setNewsCache(data []models.NewsItem) {
	newsCache.mutex.Lock()
	defer newsCache.mutex.Unlock()
	newsCache.Data = data
	// ⭐️ 1. 수정: 만료 시간을 30분으로 설정
	newsCache.ExpiresAt = time.Now().Add(30 * time.Minute)
	log.Printf("새로운 뉴스 데이터 캐시 저장 (만료 시간: %v)", newsCache.ExpiresAt)
}

// --- ⭐️ 3. 수정된 캐시 로직 함수 ---
func fetchAndCacheNews() ([]models.NewsItem, error) {
	// 캐시가 있으면 캐시 반환
	if cachedNews, ok := getNewsFromCache(); ok {
		return cachedNews, nil
	}

	// 캐시가 없으면 API 호출 함수(신규)를 호출
	result, err := fetchNewsFromAPI()
	if err != nil {
		log.Printf("뉴스 데이터 가져오기 실패: %v", err)
		return nil, err
	}

	// ⭐️ 4. 오타 수정: setCache -> setNewsCache
	// API 결과를 캐시에 저장
	setNewsCache(result)
	return result, nil
}
// --- ⭐️ 5. 수정된 핸들러 함수 ---
// (API 호출 로직은 모두 제거되고 캐시 호출 및 렌더링만 남음)
func GetTopNews(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	// 1. 캐시 로직 함수 호출
	articles, err := fetchAndCacheNews()
	if err != nil {
		http.Error(w, "뉴스 정보를 가져올 수 없습니다.", http.StatusInternalServerError)
		return
	}

	if len(articles) == 0 {
		fmt.Fprint(w, "<p>가져온 뉴스가 없습니다.</p>")
		return
	}

	// 2. HTML 렌더링 (변경 없음)
	var htmlBuilder strings.Builder
	for _, item := range articles {
		title := cleanHtmlTags(item.Title)
		description := cleanHtmlTags(item.Description)

		htmlBuilder.WriteString(fmt.Sprintf(
			`<div class="news-item">
                <h4><a href="%s" target="_blank">%s</a></h4>
                <p>%s</p>
            </div>`,
			item.Link,
			title,
			description,
		))
	}

	fmt.Fprint(w, htmlBuilder.String())
}