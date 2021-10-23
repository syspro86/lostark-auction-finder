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

type Context struct {
	// 조건 관련
	CharacterName      string
	LearnedBuffs       map[string]int // 이미 배운 각인
	SupposedStoneLevel []int          // 어빌스톤 예상 단계
	Grade              string
	AuctionItemCount   int // 옵션별 최소 몇개 경매품 검색할 지
	// 목표 관련
	Budget          int // 예산 (골드)
	TargetBuffs     map[string]int
	TargetBuffNames []string // 맞추려는 각인
	TargetLevels    []int
	TargetStats     []string // 2차 스탯
	TargetQuality   string   // "전체 품질", 10 이상, 90 이상
	OnlyFirstStat   bool
	MaxDebuffLevel  int

	TargetTripods [][]string
}

func (ctx *Context) Load(fileName string) {
	if data, err := os.ReadFile(fileName); err == nil {
		tmp := Context{}
		if err := json.Unmarshal(data, &tmp); err == nil {
			*ctx = tmp

			ctx.TargetBuffNames = make([]string, 0)
			ctx.TargetLevels = make([]int, 0)
			for name, level := range ctx.TargetBuffs {
				ctx.TargetBuffNames = append(ctx.TargetBuffNames, name)
				ctx.TargetLevels = append(ctx.TargetLevels, level*5)
			}
			ctx.OnlyFirstStat = true
		} else {
			log.Println(err)
		}
	}
}

func (ctx *Context) Save(fileName string) {
	data, _ := json.MarshalIndent(*ctx, "", "  ")
	os.WriteFile("config.json", data, 0644)
}
