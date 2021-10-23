package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"
)

type TripodItem struct {
	Name       string
	Price      int
	Buffs      []int
	Stats      []int
	Debuffs    []int
	EtcBuffs   []int
	Quality    string
	DebuffDesc string
	UniqueName bool
}

func suggestTripod() {
	//allItemList :=
	searchTripod()
}

func searchTripod() [][]TripodItem {
	weapons := make([]TripodItem, 0)
	helmets := make([]TripodItem, 0)
	bodys := make([]TripodItem, 0)
	legs := make([]TripodItem, 0)
	gloves := make([]TripodItem, 0)
	shoulders := make([]TripodItem, 0)

	steps := []string{"무기", "투구", "상의", "하의", "장갑", "어깨"}
	categories := []string{"장비 - 무기", "장비 - 투구", "장비 - 상의", "장비 - 하의", "장비 - 장갑", "장비 - 어깨"}

	characterClass, _ := getItemsFromCharacter()

	for step := 0; step < len(steps); step++ {
		dstItems := []*[]TripodItem{&weapons, &helmets, &bodys, &legs, &gloves, &shoulders}[step]
		searchResult := make([][]string, 0)
		addResult := func(src [][]string) {
			if len(src) > 0 {
				searchResult = append(searchResult, src...)
			}
		}
		for i := 0; i < len(ctx.TargetTripods); i++ {
			tripod1 := ctx.TargetTripods[i]
			addResult(readOrSearchTripodItem(categories[step], characterClass, steps[step], []string{tripod1[0]}, []string{tripod1[1]}, []string{tripod1[2]}))
			for j := i + 1; j < len(ctx.TargetTripods); j++ {
				tripod2 := ctx.TargetTripods[j]
				addResult(readOrSearchTripodItem(categories[step], characterClass, steps[step], []string{tripod1[0], tripod2[0]}, []string{tripod1[1], tripod2[1]}, []string{tripod1[2], tripod2[2]}))
				for k := j + 1; k < len(ctx.TargetTripods); k++ {
					tripod3 := ctx.TargetTripods[k]
					addResult(readOrSearchTripodItem(categories[step], characterClass, steps[step], []string{tripod1[0], tripod2[0], tripod3[0]}, []string{tripod1[1], tripod2[1], tripod3[1]}, []string{tripod1[2], tripod2[2], tripod3[2]}))
				}
			}
		}
		*dstItems = append(*dstItems, TripodItem{})
	}
	log.Println("수집 종료")

	return [][]TripodItem{weapons, helmets, bodys, legs, gloves, shoulders}
}

func readOrSearchTripodItem(category string, characterClass string, stepName string, skillNames []string, tripodNames []string, tripodLevels []string) [][]string {
	// filename
	fileName := fmt.Sprintf("트포_%s", stepName)
	for i := 0; i < len(skillNames); i++ {
		fileName += fmt.Sprintf("_%s_%s_%s", skillNames[i], tripodNames[i], tripodLevels[i])
	}
	fileName += ".json"

	// 캐시 데이터가 한시간 이상 지났으면 삭제
	if stat, err := os.Stat(toolConfig.CachePath + fileName); err == nil {
		if stat.ModTime().Add(time.Hour).Before(time.Now()) {
			os.Remove(toolConfig.FileBase + fileName)
		}
	}

	if data, err := os.ReadFile(toolConfig.CachePath + fileName); err == nil {
		tmp := new([][]string)
		json.Unmarshal(data, tmp)
		return *tmp
	} else {
		loginStove()
		openAuction()

		selectDetailOption(".lui-modal__window .select--deal-category", category)
		selectDetailOption(".lui-modal__window .select--deal-class", characterClass)
		selectDetailOption(".lui-modal__window .select--deal-itemtier", "티어 3")

		for i := 0; i < len(skillNames); i++ {
			selectEtcDetailOption(fmt.Sprintf(".lui-modal__window #selSkill_%d", i), skillNames[i])
			selectEtcDetailOption(fmt.Sprintf(".lui-modal__window #selSkillSub_%d", i), tripodNames[i])
			selectSkillMinLevel(fmt.Sprintf(".lui-modal__window #txtSkillMin_%d", i), tripodLevels[i])
		}

		progMsg := fmt.Sprintf("%s 검색", stepName)
		for i := 0; i < len(skillNames); i++ {
			progMsg += fmt.Sprintf(" [%s, %s, %s]", skillNames[i], tripodNames[i], tripodLevels[i])
		}
		log.Println(progMsg)

		ret, retry := searchAndGetResults()
		for retry {
			log.Println("1분후 재검색")
			time.Sleep(time.Minute)
			ret, retry = searchAndGetResults()
		}
		log.Printf("검색 결과 [%d]건", len(ret))
		if toolConfig.CacheSearchResult {
			data, _ := json.MarshalIndent(ret, "", "  ")
			if _, err := os.Stat(toolConfig.CachePath); errors.Is(err, os.ErrNotExist) {
				os.Mkdir(toolConfig.CachePath, 0755)
			}
			os.WriteFile(toolConfig.CachePath+fileName, data, 0644)
		}
		return ret
	}
}
