package main

import (
	"encoding/json"
	"log"
	"os"
)

type ToolConfig struct {
	FileBase           string
	SeleniumURL        string
	ChromeUserDataPath string
	CacheSearchResult  bool
	CachePath          string
	LogLevel           string
}

var toolConfig = ToolConfig{
	CacheSearchResult: true,
	CachePath:         "cache/",
}

func (cfg *ToolConfig) Load(fileName string) {
	if data, err := os.ReadFile(fileName); err == nil {
		tmp := ToolConfig{}
		if err := json.Unmarshal(data, &tmp); err == nil {
			*cfg = tmp
		} else {
			log.Println(err)
		}
	}
}

// 전투 특성
type BattleStat struct {
	Critical int
	Haste    int
	Mastery  int
}

func (bs *BattleStat) ToNames() []string {
	return []string{"치명", "신속", "특화"}
}

func (bs *BattleStat) ToInts() []int {
	return []int{bs.Critical, bs.Haste, bs.Mastery}
}

type Context struct {
	// 조건 관련
	CharacterName      string
	LearnedBuffs       map[string]int // 이미 배운 각인
	SupposedStoneLevel []int          // 어빌스톤 예상 단계
	Grade              string
	AuctionItemCount   int // 옵션별 최소 몇개 경매품 검색할 지
	// 목표 관련
	TargetBuffs       map[string]int
	TargetStats       BattleStat
	TargetQuality     string // "전체 품질", 10 이상, 90 이상
	TargetQualityNeck string // "전체 품질", 10 이상, 90 이상
	MaxDebuffLevel    int

	TargetTripods [][]string

	// 작업 관련
	ThreadCount int
}

func (ctx *Context) Load(fileName string) {
	if data, err := os.ReadFile(fileName); err == nil {
		tmp := Context{}
		if err := json.Unmarshal(data, &tmp); err == nil {
			*ctx = tmp
		} else {
			log.Println(err)
		}
	}
}

func (ctx *Context) Save(fileName string) {
	data, _ := json.MarshalIndent(*ctx, "", "  ")
	os.WriteFile("config.json", data, 0644)
}
