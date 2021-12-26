package main

import (
	"fmt"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/syspro86/lostark-auction-finder/pkg/loa"
)

type AccessoryJob struct {
	Web             *WebClient
	LogWriter       WriteFunction
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
	for name := range job.Ctx.TargetBuffs {
		job.TargetBuffNames = append(job.TargetBuffNames, name)
	}
	sort.Strings(job.TargetBuffNames)
	for _, name := range job.TargetBuffNames {
		job.TargetLevels = append(job.TargetLevels, job.Ctx.TargetBuffs[name]*5)
	}

	allItemList, comparingIndex := job.searchAccessory()
	statPossibleAdd := make([]int, len(allItemList)+1)
	for i, v := range []int{
		loa.Const.MaxStats[job.Ctx.Grade]["반지"],
		loa.Const.MaxStats[job.Ctx.Grade]["반지"],
		loa.Const.MaxStats[job.Ctx.Grade]["귀걸이"],
		loa.Const.MaxStats[job.Ctx.Grade]["귀걸이"],
		loa.Const.MaxStats[job.Ctx.Grade]["목걸이"]} {
		statPossibleAdd[len(statPossibleAdd)-i-2] = statPossibleAdd[len(statPossibleAdd)-i-1] + v
	}
	for i := range statPossibleAdd {
		if statPossibleAdd[len(statPossibleAdd)-i-1] == 0 && i > 0 {
			statPossibleAdd[len(statPossibleAdd)-i-1] = statPossibleAdd[len(statPossibleAdd)-i]
		}
	}
	log.WithField("statPossibleMax", statPossibleAdd).Infoln("전투특성 추가치")

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

	type ItemCombSet struct {
		step          int
		curPrice      int
		curPeon       int
		curBuffs      []int
		curStats      []int
		curDebuffs    []int
		itemIndexList []int
	}

	numCPU := runtime.NumCPU()
	if job.Ctx.ThreadCount > 0 {
		numCPU = job.Ctx.ThreadCount
	}
	args := make(chan ItemCombSet, numCPU+10)
	wg := sync.WaitGroup{}
	curJobCount := int32(0)

	newStepArg := func(arg ItemCombSet) {
		log.Traceln(arg)
		args <- arg
	}

	targetStats := job.Ctx.TargetStats.ToInts()
	minGold := map[int][]int{}
	minGoldLock := sync.RWMutex{}

	getMinGold := func(level int) int {
		minGoldLock.Lock()
		defer minGoldLock.Unlock()
		if v, ok := minGold[level]; ok {
			return v[len(v)-1]
		}
		minGold[level] = []int{1_000_000}
		return minGold[level][0]
	}

	setMinGold := func(level int, value int) {
		minGoldLock.Lock()
		defer minGoldLock.Unlock()
		for i := level; i <= job.Ctx.MaxDebuffLevel; i++ {
			minGold[i] = append(minGold[i], value)
			if len(minGold[i]) > 100 {
				minGold[i] = minGold[i][0:100]
			}
		}
	}

	reportResult := func(itemSet ItemCombSet) {
		getDebuffDescAll := func(arr []int) string {
			desc := ""
			for i, v := range arr {
				if v > 0 {
					if v >= 5 {
						desc += fmt.Sprintf("%s(%d) ", loa.Const.Debuffs[i], int(v/5))
					}
				}
			}
			return desc
		}

		itemList := []AccessoryItem{}
		for i, v := range itemSet.itemIndexList {
			itemList = append(itemList, allItemList[i][v])
		}
		job.LogWriter("result", map[string]interface{}{
			"price":     itemSet.curPrice,
			"peon":      itemSet.curPeon,
			"debuffs":   getDebuffDescAll(itemSet.curDebuffs),
			"stats":     itemSet.curStats,
			"buffNames": job.TargetBuffNames,
			"statNames": job.Ctx.TargetStats.ToNames(),
			"items":     itemList,
		})
	}

	var checkItemSet func(itemSet ItemCombSet)
	checkItemSet = func(itemSet ItemCombSet) {
		remainStep := len(allItemList) - itemSet.step
		for i, v := range targetStats {
			if itemSet.curStats[i]+statPossibleAdd[itemSet.step] < v {
				return
			}
		}
		if remainStep == 0 {
			debuffLevel := getDebuffLevel(itemSet.curDebuffs)
			if itemSet.curPrice >= getMinGold(debuffLevel) {
				return
			}
			setMinGold(debuffLevel, itemSet.curPrice)
			reportResult(itemSet)
			log.WithField("itemSet", itemSet).Info("New Item Set")
		} else {
			if itemSet.curPrice > getMinGold(job.Ctx.MaxDebuffLevel) {
				return
			}
			for index, item := range allItemList[itemSet.step] {
				if comparingIndex[itemSet.step] {
					if index < itemSet.itemIndexList[itemSet.step-1] {
						continue
					}
				}
				if itemSet.curPrice+item.Price > getMinGold(job.Ctx.MaxDebuffLevel) {
					continue
				}
				hasUniqueName := false
				if item.UniqueName {
					for step2, index2 := range itemSet.itemIndexList {
						if allItemList[step2][index2].Name == item.Name {
							hasUniqueName = true
							break
						}
					}
				}
				if hasUniqueName {
					continue
				}
				nextBuffs := sumArray(itemSet.curBuffs, item.Buffs)

				getMaxInsufficientPoint := func(arr []int) int {
					pt := 0
					for i, num := range arr {
						if num < job.TargetLevels[i] {
							if pt < job.TargetLevels[i]-num {
								pt = job.TargetLevels[i] - num
							}
						}
					}
					return pt
				}
				getTotalInsufficientPoint := func(arr []int) int {
					pt := 0
					for i, num := range arr {
						if num < job.TargetLevels[i] {
							pt += job.TargetLevels[i] - num
						}
					}
					return pt
				}

				if itemSet.step >= 3 && getMaxInsufficientPoint(nextBuffs) > 3*(remainStep-1) {
					continue
				}
				if itemSet.step >= 3 && getTotalInsufficientPoint(nextBuffs) > loa.Const.MaxBuffPointPerGrade[job.Ctx.Grade]*(remainStep-1) {
					continue
				}
				nextDebuffs := sumArray(itemSet.curDebuffs, item.Debuffs)
				if getDebuffLevel(nextDebuffs) > job.Ctx.MaxDebuffLevel {
					continue
				}
				nextStats := sumArray(itemSet.curStats, item.Stats)

				newItemSet := ItemCombSet{
					step:          itemSet.step + 1,
					curPrice:      itemSet.curPrice + item.Price,
					curPeon:       itemSet.curPeon + item.Peon,
					curBuffs:      nextBuffs,
					curStats:      nextStats,
					curDebuffs:    nextDebuffs,
					itemIndexList: append(itemSet.itemIndexList, index),
				}
				if curJobCount > int32(numCPU) && itemSet.step >= 2 {
					checkItemSet(newItemSet)
				} else {
					atomic.AddInt32(&curJobCount, 1)
					wg.Add(1)
					newStepArg(newItemSet)
				}
			}
		}
	}

	processItemSets := func(itemSets <-chan ItemCombSet) {
		for itemSet := range itemSets {
			checkItemSet(itemSet)
			atomic.AddInt32(&curJobCount, -1)
			wg.Done()
		}
	}

	wg.Add(1)
	newStepArg(ItemCombSet{
		step:          0,
		curPrice:      0,
		curPeon:       0,
		curBuffs:      make([]int, len(job.TargetBuffNames)),
		curStats:      make([]int, len(job.Ctx.TargetStats.ToInts())),
		curDebuffs:    make([]int, len(loa.Const.Debuffs)),
		itemIndexList: make([]int, 0),
	})

	log.Infoln("item comb started")
	for i := 0; i < numCPU; i++ {
		go processItemSets(args)
	}
	wg.Wait()
	log.Infoln("item comb finished")
	close(args)
	job.LogWriter("end", "")
}

func (job *AccessoryJob) searchAccessory() ([][]AccessoryItem, []bool) {
	log.Info("start searchAccessory")
	stoneItems := make([]AccessoryItem, 0)
	neckItems := make([]AccessoryItem, 0)
	earItems := make([]AccessoryItem, 0)
	ringItems := make([]AccessoryItem, 0)

	steps := []string{"어빌리티 스톤", "목걸이", "귀걸이", "반지"}
	categories := []string{"어빌리티 스톤 - 전체", "장신구 - 목걸이", "장신구 - 귀걸이", "장신구 - 반지"}
	qualities := []string{"전체 품질", job.Ctx.TargetQualityNeck, job.Ctx.TargetQuality, job.Ctx.TargetQuality}

	charInfo, _ := job.Web.GetItemsFromCharacter(job.Ctx.CharacterName)
	characterClass := charInfo.ClassName
	characterItems := [][][]string{charInfo.Stone, charInfo.Neck, charInfo.Ear, charInfo.Ring}
	job.Ctx.Grade = loa.Const.Grades[1]

	statNames := []string{}
	for i, v := range job.Ctx.TargetStats.ToInts() {
		if v > 0 {
			statNames = append(statNames, job.Ctx.TargetStats.ToNames()[i])
		}
	}

	logStatNames := log.WithField("statNames", statNames)
	buffLevelSets := [][]int{
		{5, 3}, {4, 3}, {3, 3}, {3, 4}, {3, 5},
	}

	for step := range steps {
		dstItems := []*[]AccessoryItem{&stoneItems, &neckItems, &earItems, &ringItems}[step]
		grade := job.Ctx.Grade
		if grade == "고대" && steps[step] == "어빌리티 스톤" {
			grade = "유물"
		}
		usedKeys := []string{}
		addToItems := func(searchResult [][]string, searchKey string, usePeon bool) {
			if searchKey != "" && arrayIndexOf(usedKeys, searchKey) >= 0 {
				return
			}
			usedKeys = append(usedKeys, searchKey)
			log.WithField("searchKey", searchKey).Infoln("new search key")
			for _, item := range searchResult {
				part := strings.Split(item[1], ";")
				eachBuff := make([]int, len(job.TargetBuffNames))
				eachStat := make([]int, len(job.Ctx.TargetStats.ToInts()))
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
					if i := arrayIndexOf(job.Ctx.TargetStats.ToNames(), rname); i >= 0 {
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
		addToItems(characterItems[step], "", false)

		// 내돌만 사용 옵션
		if step == 0 && !job.Ctx.SearchAbilityStone {
			continue
		}

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
					logStatNames.Traceln("step 0")
					addToItems(job.readOrSearchItem(categories[step], characterClass, steps[step], grade, job.TargetBuffNames[i], job.TargetBuffNames[j], 0, 0, "", "", ""))
				case 1:
					logStatNames.Traceln("step 1")
					for k, statName1 := range statNames {
						for l, statName2 := range statNames {
							if k >= l {
								continue
							}
							for _, levelSet := range buffLevelSets {
								addToItems(job.readOrSearchItem(categories[step], characterClass, steps[step], grade, job.TargetBuffNames[i], job.TargetBuffNames[j], levelSet[0], levelSet[1], statName1, statName2, qualities[step]))
								addToItems(job.readOrSearchItem(categories[step], characterClass, steps[step], grade, job.TargetBuffNames[i], "", levelSet[0], levelSet[1], statName1, statName2, qualities[step]))
								addToItems(job.readOrSearchItem(categories[step], characterClass, steps[step], grade, job.TargetBuffNames[j], "", levelSet[0], levelSet[1], statName1, statName2, qualities[step]))
							}
						}
					}
				case 2:
					logStatNames.Traceln("step 2")
					fallthrough
				case 3:
					logStatNames.Traceln("step 3")
					for _, statName := range statNames {
						for _, levelSet := range buffLevelSets {
							addToItems(job.readOrSearchItem(categories[step], characterClass, steps[step], grade, job.TargetBuffNames[i], job.TargetBuffNames[j], levelSet[0], levelSet[1], statName, "", qualities[step]))
							// addToItems(job.readOrSearchItem(categories[step], characterClass, steps[step], grade, job.TargetBuffNames[i], "", levelSet[0], levelSet[1], statName, "", qualities[step]))
							// addToItems(job.readOrSearchItem(categories[step], characterClass, steps[step], grade, job.TargetBuffNames[j], "", levelSet[0], levelSet[1], statName, "", qualities[step]))
						}
					}
				}
			}
		}
		sort.Slice(*dstItems, func(i, j int) bool {
			item1 := (*dstItems)[i]
			item2 := (*dstItems)[j]
			return item1.Peon < item2.Peon ||
				(item1.Peon == item2.Peon && item1.Price < item2.Price)
		})
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
				Stats:   make([]int, len(job.Ctx.TargetStats.ToInts())),
				Debuffs: make([]int, len(loa.Const.Debuffs)),
				Price:   0,
			})
		}
	}

	log.Infof("각인 개수: %d", len(bookBuffItems))
	log.Infof("어빌리티 스톤 개수: %d", len(stoneItems))
	log.Infof("목걸이 개수: %d", len(neckItems))
	log.Infof("귀걸이 개수: %d", len(earItems))
	log.Infof("반지 개수: %d", len(ringItems))

	return [][]AccessoryItem{
			bookBuffItems, bookBuffItems, stoneItems, neckItems, earItems, earItems, ringItems, ringItems,
		}, []bool{
			false, true, false, false, false, true, false, true,
		}
}

func (job *AccessoryJob) readOrSearchItem(category string, characterClass string, stepName string, grade string, buff1 string, buff2 string, buffLevel1 int, buffLevel2 int, stat1 string, stat2 string, quality string) ([][]string, string, bool) {
	cacheKey := fmt.Sprintf("%s_%s_%d", stepName, buff1, buffLevel1)
	if buff2 != "" {
		cacheKey += fmt.Sprintf("_%s_%d", buff2, buffLevel2)
	}
	if stat1 != "" {
		cacheKey += fmt.Sprintf("_%s", stat1)
	}
	if stat2 != "" {
		cacheKey += fmt.Sprintf("_%s", stat2)
	}

	ret := loadCacheResult(cacheKey)
	if len(ret) > 0 {
		return ret, cacheKey, true
	} else if runtime.GOOS != "windows" {
		return [][]string{}, cacheKey, true
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
			job.Web.selectEtcDetailText("#txtEtcMin_0", fmt.Sprintf("%d", buffLevel1))
			job.Web.selectEtcDetailText("#txtEtcMax_0", fmt.Sprintf("%d", buffLevel1))
		}
		if buff2 != "" {
			job.Web.selectEtcDetailOption(".lui-modal__window #selEtc_1", "각인 효과")
			job.Web.selectEtcDetailOption(".lui-modal__window #selEtcSub_1", buff2)
			job.Web.selectEtcDetailText("#txtEtcMin_1", fmt.Sprintf("%d", buffLevel2))
			job.Web.selectEtcDetailText("#txtEtcMax_1", fmt.Sprintf("%d", buffLevel2))
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
			saveCacheResult(cacheKey, ret)
		}
		return ret, cacheKey, true
	}
}
