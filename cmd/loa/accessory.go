package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/syspro86/lostark-auction-finder/pkg/loa"
	"github.com/syspro86/lostark-auction-finder/pkg/tools"
)

type AccessoryItem struct {
	Name       string
	Price      int
	Buffs      []int
	Stats      []int
	Debuffs    []int
	EtcBuffs   []int
	Quality    string
	DebuffDesc string
	UniqueName bool
	Peon       int
}

func suggestAccessory(writeLog func(string, interface{})) {
	allItemList := searchAccessory(writeLog)

	sumArray := func(srcs ...[]int) []int {
		if len(srcs) == 0 {
			return make([]int, 0)
		} else {
			dst := make([]int, len(srcs[0]))
			for _, src := range srcs {
				for i := 0; i < len(dst); i++ {
					dst[i] += src[i]
				}
			}
			return dst
		}
	}

	getDebuffLevel := func(arr []int) int {
		debuffLevel := 0
		for _, level := range arr {
			debuffLevel += level / 5
		}
		return debuffLevel
	}

	allTable := [][]string{}
	resultHeader := []string{}
	resultHeader = append(resultHeader, "세트 번호")
	resultHeader = append(resultHeader, "부위 번호")
	resultHeader = append(resultHeader, "총 골드")
	resultHeader = append(resultHeader, "총 페온")
	resultHeader = append(resultHeader, "전체 디버프")
	for _, v := range ctx.TargetStats {
		resultHeader = append(resultHeader, fmt.Sprintf("총 %s", v))
	}
	resultHeader = append(resultHeader, "부위 골드")
	resultHeader = append(resultHeader, "부위 페온")
	resultHeader = append(resultHeader, "아이템 이름")
	resultHeader = append(resultHeader, ctx.TargetBuffNames...)
	resultHeader = append(resultHeader, "퀄리티")
	resultHeader = append(resultHeader, ctx.TargetStats...)
	resultHeader = append(resultHeader, "디버프")
	writeLog("resultHeader", resultHeader)
	allTable = append(allTable, resultHeader)
	// saveExcel := func() {
	// 	file, err := os.OpenFile(tools.Config.FileBase+"result.xls", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	// 	if err == nil {
	// 		file.WriteString("<table>\n")
	// 		for idx, tr := range allTable {
	// 			if idx == 0 {
	// 				file.WriteString("<tr>\n")
	// 			} else if ((idx-1)/8)%2 == 0 {
	// 				file.WriteString("<tr bgcolor='#80ff80'>\n")
	// 			} else {
	// 				file.WriteString("<tr bgcolor='#8080ff'>\n")
	// 			}
	// 			for _, td := range tr {
	// 				file.WriteString("<td>")
	// 				file.WriteString(td)
	// 				file.WriteString("</td>\n")
	// 			}
	// 			file.WriteString("</tr>\n")
	// 		}
	// 		file.WriteString("</table>")
	// 		file.Sync()
	// 		file.Close()
	// 	}
	// }

	minGold := 1_000_000
	var checkItemSet func(step int, curPrice int, curPeon int, curBuffs []int, curStats []int, curDebuffs []int, itemIndexList []int)
	checkItemSet = func(step int, curPrice int, curPeon int, curBuffs []int, curStats []int, curDebuffs []int, itemIndexList []int) {
		remainStep := len(allItemList) - step
		if remainStep == 0 {
			if curPrice >= minGold {
				return
			}
			minGold = curPrice

			fill := func(str string, num int) string {
				totalLen := 0
				for _, v := range str {
					if v >= 256 {
						totalLen += 2
					} else {
						totalLen++
					}
				}
				for ; totalLen < num; totalLen++ {
					str = " " + str
				}
				return str
			}

			getDebuffDescAll := func(arr []int) string {
				desc := ""
				for i, v := range arr {
					if v > 0 {
						if v > 5 {
							desc += fmt.Sprintf("%s(%d) ", loa.Const.Debuffs[i], int(v/5))
						}
					}
				}
				return desc
			}

			statDescHeader := ""
			for i, v := range ctx.TargetStats {
				statDescHeader += fmt.Sprintf("%s %d ", v, curStats[i])
			}
			buffHeader := ""
			for _, v := range ctx.TargetBuffNames {
				buffHeader += fmt.Sprintf("%s ", fill(v, 20))
			}
			statHeader := ""
			for _, v := range ctx.TargetStats {
				statHeader += fmt.Sprintf("%s ", fill(v, 6))
			}

			for i, v := range itemIndexList {
				resultRow := []string{}
				resultRow = append(resultRow, fmt.Sprintf("%d", (len(allTable)+len(itemIndexList)-1)/len(itemIndexList)))
				resultRow = append(resultRow, fmt.Sprintf("%d", i+1))
				resultRow = append(resultRow, fmt.Sprintf("%d", curPrice))
				resultRow = append(resultRow, fmt.Sprintf("%d", curPeon))
				resultRow = append(resultRow, getDebuffDescAll(curDebuffs))
				for _, v := range curStats {
					resultRow = append(resultRow, fmt.Sprintf("%d", v))
				}
				resultRow = append(resultRow, fmt.Sprintf("%d", allItemList[i][v].Price))
				resultRow = append(resultRow, fmt.Sprintf("%d", allItemList[i][v].Peon))
				resultRow = append(resultRow, allItemList[i][v].Name)
				for _, v := range allItemList[i][v].Buffs {
					resultRow = append(resultRow, fmt.Sprintf("%d", v))
				}
				resultRow = append(resultRow, allItemList[i][v].Quality)
				for _, v := range allItemList[i][v].Stats {
					resultRow = append(resultRow, fmt.Sprintf("%d", v))
				}
				resultRow = append(resultRow, allItemList[i][v].DebuffDesc)
				allTable = append(allTable, resultRow)

				writeLog("result", map[string]interface{}{
					"price":   curPrice,
					"peon":    curPeon,
					"debuffs": getDebuffDescAll(curDebuffs),
					"stats":   curStats,
				})
				// saveExcel()
			}

			msg := fmt.Sprintf("(골드) %d (페온) %d (디버프) %s (스탯 합) %s\n", curPrice, curPeon, getDebuffDescAll(curDebuffs), statDescHeader)
			msg += fmt.Sprintf(" %s%s%s%s\n",
				buffHeader,
				fill("GOLD", 6),
				statHeader,
				fill("품질", 6))
			for i, v := range itemIndexList {
				buffStr := ""
				for z := range ctx.TargetBuffNames {
					buffStr += fmt.Sprintf("%20d ", allItemList[i][v].Buffs[z])
				}
				statStr := ""
				for z := range ctx.TargetStats {
					statStr += fmt.Sprintf("%6d ", allItemList[i][v].Stats[z])
				}
				msg += fmt.Sprintf(" %s%6d%s%s (%s) %s\n",
					buffStr,
					allItemList[i][v].Price,
					statStr,
					fill(allItemList[i][v].Quality, 6),
					fill(allItemList[i][v].DebuffDesc, 20),
					allItemList[i][v].Name,
				)
			}
			msg += "\n"
			log.Println(msg)
			// writeLog("log", msg)

			return
		}

		for index, item := range allItemList[step] {
			if curPrice+item.Price > ctx.Budget || curPrice+item.Price > minGold {
				continue
			}
			hasUniqueName := false
			if item.UniqueName {
				for step2, index2 := range itemIndexList {
					if allItemList[step2][index2].Name == item.Name {
						hasUniqueName = true
						break
					}
				}
			}
			if hasUniqueName {
				continue
			}
			nextBuffs := sumArray(curBuffs, item.Buffs)

			getInsufficientPoint := func(arr []int) int {
				pt := 0
				for i, num := range arr {
					if num < ctx.TargetLevels[i] {
						pt += ctx.TargetLevels[i] - num
					}
				}
				return pt
			}

			if step >= 3 && getInsufficientPoint(nextBuffs) > loa.Const.MaxBuffPointPerGrade[ctx.Grade]*(remainStep-1) {
				continue
			}
			nextDebuffs := sumArray(curDebuffs, item.Debuffs)
			if getDebuffLevel(nextDebuffs) > ctx.MaxDebuffLevel {
				continue
			}
			nextStats := sumArray(curStats, item.Stats)

			checkItemSet(step+1, curPrice+item.Price, curPeon+item.Peon, nextBuffs, nextStats, nextDebuffs, append(itemIndexList, index))
		}
	}

	checkItemSet(0, 0, 0,
		make([]int, len(ctx.TargetBuffNames)),
		make([]int, len(ctx.TargetStats)),
		make([]int, len(loa.Const.Debuffs)),
		make([]int, 0),
	)
	writeLog("end", "")
}

func searchAccessory(writeLog func(string, interface{})) [][]AccessoryItem {
	stoneItems := make([]AccessoryItem, 0)
	neckItems := make([]AccessoryItem, 0)
	earItems := make([]AccessoryItem, 0)
	ringItems := make([]AccessoryItem, 0)

	steps := []string{"어빌리티 스톤", "목걸이", "귀걸이", "반지"}
	categories := []string{"어빌리티 스톤 - 전체", "장신구 - 목걸이", "장신구 - 귀걸이", "장신구 - 반지"}
	qualities := []string{"전체 품질", ctx.TargetQuality, ctx.TargetQuality, ctx.TargetQuality}

	characterClass, characterItems := getItemsFromCharacter()

	for step := range steps {
		dstItems := []*[]AccessoryItem{&stoneItems, &neckItems, &earItems, &ringItems}[step]
		grade := ctx.Grade
		if grade == "고대" && steps[step] == "어빌리티 스톤" {
			grade = "유물"
		}
		addToItems := func(searchResult [][]string, usePeon bool) {
			for _, item := range searchResult {
				part := strings.Split(item[1], ";")
				eachBuff := make([]int, len(ctx.TargetBuffNames))
				eachStat := make([]int, len(ctx.TargetStats))
				eachDebuff := make([]int, len(loa.Const.Debuffs))
				eachQuality := ""
				if len(item) >= 4 {
					eachQuality = item[3]
				}
				supposedLevel := ctx.SupposedStoneLevel[:]
				for _, p := range part {
					rname := strings.Split(p, "]")[0][1:]
					rlevel := 0
					if strings.Contains(p, "세공기회") {
						rlevel = supposedLevel[0]
						supposedLevel = supposedLevel[1:]
					} else {
						rlevel = parseInt(strings.Split(p, "+")[1])
					}
					if i := arrayIndexOf(ctx.TargetBuffNames, rname); i >= 0 {
						eachBuff[i] = rlevel
					}
					if i := arrayIndexOf(ctx.TargetStats, rname); i >= 0 {
						eachStat[i] = rlevel
					}
					if i := arrayIndexOf(loa.Const.Debuffs, rname); i >= 0 {
						eachDebuff[i] = rlevel
					}
				}
				getDebuffDesc := func(arr []int) string {
					for i, v := range arr {
						if v > 0 {
							return fmt.Sprintf("%s %d", loa.Const.Debuffs[i], v)
						}
					}
					return ""
				}
				peon := 0
				if usePeon {
					peon = loa.Const.Peons[grade][steps[step]]
				}
				*dstItems = append(*dstItems, AccessoryItem{
					Name:       item[0],
					Buffs:      eachBuff,
					Stats:      eachStat,
					Debuffs:    eachDebuff,
					Price:      parseInt(item[2]),
					Quality:    eachQuality,
					DebuffDesc: getDebuffDesc(eachDebuff),
					UniqueName: true,
					Peon:       peon,
				})
			}
		}
		addToItems(characterItems[step], false)

		for i := 0; i < len(ctx.TargetBuffNames); i++ {
			for j := i + 1; j < len(ctx.TargetBuffNames); j++ {
				if step == 0 {
					if arrayIndexOf(loa.Const.ClassBuffs, ctx.TargetBuffNames[i]) >= 0 || arrayIndexOf(loa.Const.ClassBuffs, ctx.TargetBuffNames[j]) >= 0 {
						continue
					}
				}
				switch step {
				case 0:
					addToItems(readOrSearchItem(writeLog, categories[step], characterClass, steps[step], grade, ctx.TargetBuffNames[i], ctx.TargetBuffNames[j], "", "", ""), true)
				case 1:
					addToItems(readOrSearchItem(writeLog, categories[step], characterClass, steps[step], grade, ctx.TargetBuffNames[i], ctx.TargetBuffNames[j], ctx.TargetStats[0], ctx.TargetStats[1], qualities[step]), true)
					addToItems(readOrSearchItem(writeLog, categories[step], characterClass, steps[step], grade, ctx.TargetBuffNames[i], "", ctx.TargetStats[0], ctx.TargetStats[1], qualities[step]), true)
					addToItems(readOrSearchItem(writeLog, categories[step], characterClass, steps[step], grade, ctx.TargetBuffNames[j], "", ctx.TargetStats[0], ctx.TargetStats[1], qualities[step]), true)
				case 2:
					fallthrough
				case 3:
					addToItems(readOrSearchItem(writeLog, categories[step], characterClass, steps[step], grade, ctx.TargetBuffNames[i], ctx.TargetBuffNames[j], ctx.TargetStats[0], "", qualities[step]), true)
					addToItems(readOrSearchItem(writeLog, categories[step], characterClass, steps[step], grade, ctx.TargetBuffNames[i], "", ctx.TargetStats[0], "", qualities[step]), true)
					addToItems(readOrSearchItem(writeLog, categories[step], characterClass, steps[step], grade, ctx.TargetBuffNames[j], "", ctx.TargetStats[0], "", qualities[step]), true)
					if !ctx.OnlyFirstStat {
						addToItems(readOrSearchItem(writeLog, categories[step], characterClass, steps[step], grade, ctx.TargetBuffNames[i], ctx.TargetBuffNames[j], ctx.TargetStats[1], "", qualities[step]), true)
						addToItems(readOrSearchItem(writeLog, categories[step], characterClass, steps[step], grade, ctx.TargetBuffNames[i], "", ctx.TargetStats[1], "", qualities[step]), true)
						addToItems(readOrSearchItem(writeLog, categories[step], characterClass, steps[step], grade, ctx.TargetBuffNames[j], "", ctx.TargetStats[1], "", qualities[step]), true)
					}
				}
			}
		}
	}
	writeLog("log", "수집 종료")

	bookBuffItems := make([]AccessoryItem, 0)
	for name, leanBuffLevel := range ctx.LearnedBuffs {
		index := arrayIndexOf(ctx.TargetBuffNames, name)
		if index >= 0 {
			buffs := make([]int, len(ctx.TargetBuffNames))
			buffs[index] = leanBuffLevel
			bookBuffItems = append(bookBuffItems, AccessoryItem{
				Name:    "[각인]" + name,
				Buffs:   buffs,
				Stats:   make([]int, len(ctx.TargetStats)),
				Debuffs: make([]int, len(loa.Const.Debuffs)),
				Price:   0,
			})
		}
	}

	return [][]AccessoryItem{
		bookBuffItems, bookBuffItems, stoneItems, neckItems, earItems, earItems, ringItems, ringItems,
	}
}

func readOrSearchItem(writeLog func(string, interface{}), category string, characterClass string, stepName string, grade string, buff1 string, buff2 string, stat1 string, stat2 string, quality string) [][]string {
	// filename
	fileName := fmt.Sprintf("%s_%s", stepName, buff1)
	if buff2 != "" {
		fileName += fmt.Sprintf("_%s", buff2)
	}
	if stat1 != "" {
		fileName += fmt.Sprintf("_%s", stat1)
	}
	if stat2 != "" {
		fileName += fmt.Sprintf("_%s", stat2)
	}
	fileName += ".json"

	// 캐시 데이터가 한시간 이상 지났으면 삭제
	if stat, err := os.Stat(tools.Config.CachePath + fileName); err == nil {
		if stat.ModTime().Add(time.Hour * 24).Before(time.Now()) {
			os.Remove(tools.Config.FileBase + fileName)
		}
	}

	if data, err := os.ReadFile(tools.Config.CachePath + fileName); err == nil {
		tmp := new([][]string)
		json.Unmarshal(data, tmp)
		return *tmp
	} else {
		loginStove()
		openAuction()

		selectDetailOption(".lui-modal__window .select--deal-category", category)
		selectDetailOption(".lui-modal__window .select--deal-class", characterClass)
		selectDetailOption(".lui-modal__window .select--deal-grade", grade)
		selectDetailOption(".lui-modal__window .select--deal-itemtier", loa.Const.Tier)
		if quality != "" {
			selectDetailOption(".lui-modal__window .select--deal-quality", quality)
		}
		if buff1 != "" {
			selectEtcDetailOption(".lui-modal__window #selEtc_0", "각인 효과")
			selectEtcDetailOption(".lui-modal__window #selEtcSub_0", buff1)
		}
		if buff2 != "" {
			selectEtcDetailOption(".lui-modal__window #selEtc_1", "각인 효과")
			selectEtcDetailOption(".lui-modal__window #selEtcSub_1", buff2)
		}
		if stat1 != "" {
			selectEtcDetailOption(".lui-modal__window #selEtc_2", "전투 특성")
			selectEtcDetailOption(".lui-modal__window #selEtcSub_2", stat1)
		}
		if stat2 != "" {
			selectEtcDetailOption(".lui-modal__window #selEtc_3", "전투 특성")
			selectEtcDetailOption(".lui-modal__window #selEtcSub_3", stat2)
		}

		progMsg := fmt.Sprintf("%s 검색", stepName)
		if buff1 != "" {
			if buff2 == "" {
				progMsg += fmt.Sprintf(" [%s]", buff1)
			} else {
				progMsg += fmt.Sprintf(" [%s, %s]", buff1, buff2)
			}
		}
		if stat1 != "" {
			if stat2 == "" {
				progMsg += fmt.Sprintf(" [%s]", stat1)
			} else {
				progMsg += fmt.Sprintf(" [%s, %s]", stat1, stat2)
			}
		}
		writeLog("log", progMsg)

		ret, retry := searchAndGetResults()
		for retry {
			writeLog("log", "1분후 재검색")
			time.Sleep(time.Minute)
			ret, retry = searchAndGetResults()
		}
		writeLog("log", fmt.Sprintf("검색 결과 [%d]건", len(ret)))
		if tools.Config.CacheSearchResult {
			data, _ := json.MarshalIndent(ret, "", "  ")
			if _, err := os.Stat(tools.Config.CachePath); errors.Is(err, os.ErrNotExist) {
				os.Mkdir(tools.Config.CachePath, 0755)
			}
			os.WriteFile(tools.Config.CachePath+fileName, data, 0644)
		}
		return ret
	}
}
