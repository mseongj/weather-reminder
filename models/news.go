package models

// NaverNewsResponse는 Naver 뉴스 API 응답 전체 구조입니다.
type NaverNewsResponse struct {
	LastBuildDate string     `json:"lastBuildDate"`
	Total         int        `json:"total"`
	Start         int        `json:"start"`
	Display       int        `json:"display"`
	Items         []NewsItem `json:"items"` // 기사 목록
}

// NewsItem은 개별 뉴스 기사 항목입니다.
type NewsItem struct {
	Title        string `json:"title"`       // 기사 제목
	OriginalLink string `json:"originallink"` // 원문 URL
	Link         string `json:"link"`         // Naver 뉴스 URL
	Description  string `json:"description"`  // 요약
	PubDate      string `json:"pubDate"`      // 발행일
}
