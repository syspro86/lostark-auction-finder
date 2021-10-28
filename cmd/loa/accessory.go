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
)

type AccessoryJob struct {
	Web             WebClient
	LogWriter       func(string, interface{})
	Ctx             Context
	TargetBuffNames []string
	TargetLevels    []int
}

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

func (job *AccessoryJob) Start() {
	job.TargetBuffNames = make([]string, 0)
	job.TargetLevels = make([]int, 0)
	for name, level := range job.Ctx.TargetBuffs {
		job.TargetBuffNames = append(job.TargetBuffNames, name)
		job.TargetLevels = append(job.TargetLevels, level*5)
	}

	allItemList := job.searchAccessory()

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
	for _, v := range job.Ctx.TargetStats {
		resultHeader = append(resultHeader, fmt.Sprintf("총 %s", v))
	}
	resultHeader = append(resultHeader, "부위 골드")
	resultHeader = append(resultHeader, "부위 페온")
	resultHeader = append(resultHeader, "아이템 이름")
	resultHeader = append(resultHeader, job.TargetBuffNames...)
	resultHeader = append(resultHeader, "퀄리티")
	resultHeader = append(resultHeader, job.Ctx.TargetStats...)
	resultHeader = append(resultHeader, "디버프")
	job.LogWriter("resultHeader", resultHeader)
	allTable = append(allTable, resultHeader)
	// saveExcel := func() {
	// 	file, err := os.OpenFile(toolConfig.FileBase+"result.xls", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
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
			for i, v := range job.Ctx.TargetStats {
				statDescHeader += fmt.Sprintf("%s %d ", v, curStats[i])
			}
			buffHeader := ""
			for _, v := range job.TargetBuffNames {
				buffHeader += fmt.Sprintf("%s ", fill(v, 20))
			}
			statHeader := ""
			for _, v := range job.Ctx.TargetStats {
				statHeader += fmt.Sprintf("%s ", fill(v, 6))
			}

			itemList := []AccessoryItem{}
			for i, v := range itemIndexList {
				resultRow := []string{}
				resultRow = append(resultRow, fmt.Sprintf("%d", (len(allTable)+len(itemIndexList)-1)/len(itemIndexList)))
				resultRow = append(resultRow, fmt.Sprintf("%d", i+1))
				resultRow = append(resultRow, fmt.Sprintf("%d", curPrice))
				resultRow = append(resultRow, fmt.Sprintf("%d", curPeon))
				resultRow = append(resultRow, getDebuffDescAll(curDebuffs))
				for _, vv := range curStats {
					resultRow = append(resultRow, fmt.Sprintf("%d", vv))
				}
				resultRow = append(resultRow, fmt.Sprintf("%d", allItemList[i][v].Price))
				resultRow = append(resultRow, fmt.Sprintf("%d", allItemList[i][v].Peon))
				resultRow = append(resultRow, allItemList[i][v].Name)
				for _, vv := range allItemList[i][v].Buffs {
					resultRow = append(resultRow, fmt.Sprintf("%d", vv))
				}
				resultRow = append(resultRow, allItemList[i][v].Quality)
				for _, vv := range allItemList[i][v].Stats {
					resultRow = append(resultRow, fmt.Sprintf("%d", vv))
				}
				resultRow = append(resultRow, allItemList[i][v].DebuffDesc)
				allTable = append(allTable, resultRow)

				itemList = append(itemList, allItemList[i][v])
				// saveExcel()
			}
			job.LogWriter("result", map[string]interface{}{
				"price":     curPrice,
				"peon":      curPeon,
				"debuffs":   getDebuffDescAll(curDebuffs),
				"stats":     curStats,
				"buffNames": job.TargetBuffNames,
				"statNames": job.Ctx.TargetStats,
				"items":     itemList,
			})

			msg := fmt.Sprintf("(골드) %d (페온) %d (디버프) %s (스탯 합) %s\n", curPrice, curPeon, getDebuffDescAll(curDebuffs), statDescHeader)
			msg += fmt.Sprintf(" %s%s%s%s\n",
				buffHeader,
				fill("GOLD", 6),
				statHeader,
				fill("품질", 6))
			for i, v := range itemIndexList {
				buffStr := ""
				for z := range job.TargetBuffNames {
					buffStr += fmt.Sprintf("%20d ", allItemList[i][v].Buffs[z])
				}
				statStr := ""
				for z := range job.Ctx.TargetStats {
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
			// saveExcel()

			return
		}

		for index, item := range allItemList[step] {
			if curPrice+item.Price > minGold {
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
					if num < job.TargetLevels[i] {
						pt += job.TargetLevels[i] - num
					}
				}
				return pt
			}

			if step >= 3 && getInsufficientPoint(nextBuffs) > loa.Const.MaxBuffPointPerGrade[job.Ctx.Grade]*(remainStep-1) {
				continue
			}
			nextDebuffs := sumArray(curDebuffs, item.Debuffs)
			if getDebuffLevel(nextDebuffs) > job.Ctx.MaxDebuffLevel {
				continue
			}
			nextStats := sumArray(curStats, item.Stats)

			checkItemSet(step+1, curPrice+item.Price, curPeon+item.Peon, nextBuffs, nextStats, nextDebuffs, append(itemIndexList, index))
		}
	}

	checkItemSet(0, 0, 0,
		make([]int, len(job.TargetBuffNames)),
		make([]int, len(job.Ctx.TargetStats)),
		make([]int, len(loa.Const.Debuffs)),
		make([]int, 0),
	)
	job.LogWriter("end", "")
}

func (job *AccessoryJob) searchAccessory() [][]AccessoryItem {
	stoneItems := make([]AccessoryItem, 0)
	neckItems := make([]AccessoryItem, 0)
	earItems := make([]AccessoryItem, 0)
	ringItems := make([]AccessoryItem, 0)

	steps := []string{"어빌리티 스톤", "목걸이", "귀걸이", "반지"}
	categories := []string{"어빌리티 스톤 - 전체", "장신구 - 목걸이", "장신구 - 귀걸이", "장신구 - 반지"}
	qualities := []string{"전체 품질", job.Ctx.TargetQuality, job.Ctx.TargetQuality, job.Ctx.TargetQuality}

	characterClass, characterItems := job.Web.getItemsFromCharacter(job.Ctx.CharacterName)
	job.Ctx.Grade = loa.Const.Grades[1]

	for step := range steps {
		dstItems := []*[]AccessoryItem{&stoneItems, &neckItems, &earItems, &ringItems}[step]
		grade := job.Ctx.Grade
		if grade == "고대" && steps[step] == "어빌리티 스톤" {
			grade = "유물"
		}
		addToItems := func(searchResult [][]string, usePeon bool) {
			for _, item := range searchResult {
				part := strings.Split(item[1], ";")
				eachBuff := make([]int, len(job.TargetBuffNames))
				eachStat := make([]int, len(job.Ctx.TargetStats))
				eachDebuff := make([]int, len(loa.Const.Debuffs))
				eachQuality := ""
				if len(item) >= 4 {
					eachQuality = item[3]
				}
				supposedLevel := job.Ctx.SupposedStoneLevel[:]
				for _, p := range part {
					rname := strings.Split(p, "]")[0][1:]
					rlevel := 0
					if strings.Contains(p, "세공기회") {
						rlevel = supposedLevel[0]
						supposedLevel = supposedLevel[1:]
					} else {
						rlevel = parseInt(strings.Split(p, "+")[1])
					}
					if i := arrayIndexOf(job.TargetBuffNames, rname); i >= 0 {
						eachBuff[i] = rlevel
					}
					if i := arrayIndexOf(job.Ctx.TargetStats, rname); i >= 0 {
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

		for i := 0; i < len(job.TargetBuffNames); i++ {
			for j := i + 1; j < len(job.TargetBuffNames); j++ {
				if step == 0 {
					if arrayIndexOf(loa.Const.ClassBuffs, job.TargetBuffNames[i]) >= 0 || arrayIndexOf(loa.Const.ClassBuffs, job.TargetBuffNames[j]) >= 0 {
						continue
					}
					if job.TargetBuffNames[i] > job.TargetBuffNames[j] {
						continue
					}
				}
				switch step {
				case 0:
					addToItems(job.readOrSearchItem(categories[step], characterClass, steps[step], grade, job.TargetBuffNames[i], job.TargetBuffNames[j], "", "", ""), true)
				case 1:
					addToItems(job.readOrSearchItem(categories[step], characterClass, steps[step], grade, job.TargetBuffNames[i], job.TargetBuffNames[j], job.Ctx.TargetStats[0], job.Ctx.TargetStats[1], qualities[step]), true)
					addToItems(job.readOrSearchItem(categories[step], characterClass, steps[step], grade, job.TargetBuffNames[i], "", job.Ctx.TargetStats[0], job.Ctx.TargetStats[1], qualities[step]), true)
					addToItems(job.readOrSearchItem(categories[step], characterClass, steps[step], grade, job.TargetBuffNames[j], "", job.Ctx.TargetStats[0], job.Ctx.TargetStats[1], qualities[step]), true)
				case 2:
					fallthrough
				case 3:
					addToItems(job.readOrSearchItem(categories[step], characterClass, steps[step], grade, job.TargetBuffNames[i], job.TargetBuffNames[j], job.Ctx.TargetStats[0], "", qualities[step]), true)
					addToItems(job.readOrSearchItem(categories[step], characterClass, steps[step], grade, job.TargetBuffNames[i], "", job.Ctx.TargetStats[0], "", qualities[step]), true)
					addToItems(job.readOrSearchItem(categories[step], characterClass, steps[step], grade, job.TargetBuffNames[j], "", job.Ctx.TargetStats[0], "", qualities[step]), true)
					if !job.Ctx.OnlyFirstStat {
						addToItems(job.readOrSearchItem(categories[step], characterClass, steps[step], grade, job.TargetBuffNames[i], job.TargetBuffNames[j], job.Ctx.TargetStats[1], "", qualities[step]), true)
						addToItems(job.readOrSearchItem(categories[step], characterClass, steps[step], grade, job.TargetBuffNames[i], "", job.Ctx.TargetStats[1], "", qualities[step]), true)
						addToItems(job.readOrSearchItem(categories[step], characterClass, steps[step], grade, job.TargetBuffNames[j], "", job.Ctx.TargetStats[1], "", qualities[step]), true)
					}
				}
			}
		}
	}
	job.LogWriter("log", "수집 종료")

	bookBuffItems := make([]AccessoryItem, 0)
	for name, leanBuffLevel := range job.Ctx.LearnedBuffs {
		index := arrayIndexOf(job.TargetBuffNames, name)
		if index >= 0 {
			buffs := make([]int, len(job.TargetBuffNames))
			buffs[index] = leanBuffLevel
			bookBuffItems = append(bookBuffItems, AccessoryItem{
				Name:    "[각인]" + name,
				Buffs:   buffs,
				Stats:   make([]int, len(job.Ctx.TargetStats)),
				Debuffs: make([]int, len(loa.Const.Debuffs)),
				Price:   0,
			})
		}
	}

	return [][]AccessoryItem{
		bookBuffItems, bookBuffItems, stoneItems, neckItems, earItems, earItems, ringItems, ringItems,
	}
}

func (job *AccessoryJob) readOrSearchItem(category string, characterClass string, stepName string, grade string, buff1 string, buff2 string, stat1 string, stat2 string, quality string) [][]string {
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
	if stat, err := os.Stat(toolConfig.CachePath + fileName); err == nil {
		if stat.ModTime().Add(time.Hour * 24).Before(time.Now()) {
			//os.Remove(toolConfig.FileBase + fileName)
		}
	}

	if data, err := os.ReadFile(toolConfig.CachePath + fileName); err == nil {
		tmp := new([][]string)
		json.Unmarshal(data, tmp)
		return *tmp
	} else {
		job.Web.loginStove()
		job.Web.openAuction()

		job.Web.selectDetailOption(".lui-modal__window .select--deal-category", category)
		job.Web.selectDetailOption(".lui-modal__window .select--deal-class", characterClass)
		job.Web.selectDetailOption(".lui-modal__window .select--deal-grade", grade)
		job.Web.selectDetailOption(".lui-modal__window .select--deal-itemtier", loa.Const.Tier)
		if quality != "" {
			job.Web.selectDetailOption(".lui-modal__window .select--deal-quality", quality)
		}
		if buff1 != "" {
			job.Web.selectEtcDetailOption(".lui-modal__window #selEtc_0", "각인 효과")
			job.Web.selectEtcDetailOption(".lui-modal__window #selEtcSub_0", buff1)
		}
		if buff2 != "" {
			job.Web.selectEtcDetailOption(".lui-modal__window #selEtc_1", "각인 효과")
			job.Web.selectEtcDetailOption(".lui-modal__window #selEtcSub_1", buff2)
		}
		if stat1 != "" {
			job.Web.selectEtcDetailOption(".lui-modal__window #selEtc_2", "전투 특성")
			job.Web.selectEtcDetailOption(".lui-modal__window #selEtcSub_2", stat1)
		}
		if stat2 != "" {
			job.Web.selectEtcDetailOption(".lui-modal__window #selEtc_3", "전투 특성")
			job.Web.selectEtcDetailOption(".lui-modal__window #selEtcSub_3", stat2)
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
		job.LogWriter("log", progMsg)

		ret, retry := job.Web.searchAndGetResults(job.Ctx.AuctionItemCount)
		for retry {
			job.LogWriter("log", "1분후 재검색")
			time.Sleep(time.Minute)
			ret, retry = job.Web.searchAndGetResults(job.Ctx.AuctionItemCount)
		}
		job.LogWriter("log", fmt.Sprintf("검색 결과 [%d]건", len(ret)))
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
