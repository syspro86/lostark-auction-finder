package loa

type GameConst struct {
	Buffs                []string
	ClassBuffs           []string
	Debuffs              []string
	MaxBuffPointPerGrade map[string]int
	Peons                map[string]map[string]int
	Tier                 string
}

var Const = GameConst{
	Buffs: []string{
		"각성", "강령술", "강화 방패", "결투의 대가", "구슬동자",
		"굳은 의지", "급소 타격", "기습의 대가", "긴급구조", "달인의 저력",
		"돌격대장", "마나 효율 증가", "마나의 흐름", "바리케이드", "번개의 분노",
		"부러진 뼈", "분쇄의 주먹", "불굴", "선수필승", "속전속결",
		"슈퍼 차지", "승부사", "시선 집중", "실드 관통", "아드레날린",
		"안정된 상태", "약자 무시", "에테르 포식자", "여신의 가호", "예리한 둔기",
		"원한", "위기 모면", "저주받은 인형", "전문의", "정기 흡수",
		"정밀 단도", "중갑 착용", "질량 증가", "최대 마나 증가", "추진력",
		"타격의 대가", "탈출의 명수", "폭발물 전문가",
	},
	ClassBuffs: []string{
		"갈증", "강화 무기", "고독한 기사", "광기", "광전사의 비기",
		"극의: 체술", "넘치는 교감", "달의 소리", "두 번째 동료", "멈출 수 없는 충동",
		"버스트", "분노의 망치", "사냥의 시간", "상급 소환사", "세맥타동",
		"심판자", "아르데타인의 기술", "역천지체", "오의 강화", "오의난무",
		"완벽한 억제", "일격필살", "잔재된 기운", "전투 태세", "절실한 구원",
		"절정", "절제", "점화", "죽음의 습격", "중력 수련",
		"진실된 용맹", "진화의 유산", "초심", "축복의 오라", "충격 단련",
		"포격 강화", "피스메이커", "핸드거너", "화력 강화", "환류",
		"황제의 칙령", "황후의 은총",
	},
	Debuffs: []string{
		"방어력 감소",
		"공격력 감소",
		"공격속도 감소",
		"이동속도 감소",
	},
	MaxBuffPointPerGrade: map[string]int{
		"전설": 6, "유물": 8, "고대": 9,
	},
	Peons: map[string]map[string]int{
		"전설": {"어빌리티 스톤": 5, "목걸이": 15, "귀걸이": 15, "반지": 15},
		"유물": {"어빌리티 스톤": 9, "목걸이": 25, "귀걸이": 25, "반지": 35},
		"고대": {"어빌리티 스톤": 0, "목걸이": 35, "귀걸이": 25, "반지": 35},
	},
	Tier: "티어 3",
}
