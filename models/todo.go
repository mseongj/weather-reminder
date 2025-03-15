package models

type Todo struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

type WeatherREQ struct {
	ServiceKey	string
	PageNo			int
	NumOfRows		int
	DataType		string
	base_date		string
	base_time		string
	nx					int
	ny					int
}

type WeatherRES struct {
	
}


// 구분	행정구역코드	1단계	2단계	3단계	격자 X	격자 Y	경도(시)	경도(분)	경도(초)	위도(시)	위도(분)	위도(초)	경도(초/100)	위도(초/100)
// kor	2729062800	대구광역시	달서구	도원동	88	89	128	32	3.84	35	48	16.08	128.5344	35.8044666666666		
