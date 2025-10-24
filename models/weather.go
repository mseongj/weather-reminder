package models

// WeatherResponse는 기상청 API 응답 JSON 구조체입니다.
type WeatherResponse struct {
	Response struct {
		Header struct {
			ResultCode string `json:"resultCode"`
			ResultMsg  string `json:"resultMsg"`
		} `json:"header"`
		Body struct {
			DataType string `json:"dataType"`
			Items    struct {
				Item []struct {
					BaseDate  string `json:"baseDate"`
					BaseTime  string `json:"baseTime"`
					Category  string `json:"category"`
					FcstDate  string `json:"fcstDate"`
					FcstTime  string `json:"fcstTime"`
					FcstValue string `json:"fcstValue"`
					Nx        int    `json:"nx"`
					Ny        int    `json:"ny"`
				} `json:"item"`
			} `json:"items"`
			PageNo     int `json:"pageNo"`
			NumOfRows  int `json:"numOfRows"`
			TotalCount int `json:"totalCount"`
		} `json:"body"`
	} `json:"response"`
}

// 리턴 될 변수의 struct
type WeatherItemToReturn struct {
	Date     string
	Time     string
	Category string
	Value    string
}

// WeatherItem은 파싱된 날씨 데이터를 담는 구조체입니다.
type WeatherItem struct {
	Date     string
	Time     string
	Sky      string // 하늘 상태 (맑음, 구름많음, 흐림)
	Pty      string // 강수 형태 (없음, 비, 눈 등)
	Tmp      string // 기온 (℃)
	Pop      string // 강수 확률 (%)
	Humidity string // 습도 (%)
}

// 구분	행정구역코드	1단계	2단계	3단계	격자 X	격자 Y	경도(시)	경도(분)	경도(초)	위도(시)	위도(분)	위도(초)	경도(초/100)	위도(초/100)
// kor	2729062800	대구광역시	달서구	도원동	88	89	128	32	3.84	35	48	16.08	128.5344	35.8044666666666		
