package main

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/tebeka/selenium"
)

func getItemsFromCharacter() (string, [][][]string) {
	driver := getWebDriver()
	for {
		driver.SetImplicitWaitTimeout(10 * time.Second)
		err := driver.Get(fmt.Sprintf("https://lostark.game.onstove.com/Profile/Character/%s", ctx.CharacterName))
		panicIfError(err)

		sleepShortly()
		success, err := driver.ExecuteScript(`
			var profile = document.querySelectorAll("#profile-ability script");
			if (profile.length == 0) {
				return "";
			}
			return profile[0].innerText
		`, nil)
		panicIfError(err)
		if success == "" {
			sleepShortly()
			continue
		}

		profileJson := success.(string)
		profileJson = profileJson[strings.Index(profileJson, "{"):]
		profileJson = profileJson[:strings.LastIndex(profileJson, "}")+1]
		// os.WriteFile("character.json", []byte(profileJson), 0644)

		for {
			var mapp map[string]interface{}
			err := json.Unmarshal([]byte(profileJson), &mapp)
			panicIfError(err)

			propertyList := map[string]string{}

			var visitInfo func(path string, m map[string]interface{})
			visitInfo = func(path string, m map[string]interface{}) {
				for key, value := range m {
					switch reflect.TypeOf(value).String() {
					case "string":
						propertyList[fmt.Sprintf("%s.%s", path, key)] = value.(string)
					case "bool":
						propertyList[fmt.Sprintf("%s.%s", path, key)] = fmt.Sprintf("%t", value.(bool))
					case "float64":
						propertyList[fmt.Sprintf("%s.%s", path, key)] = fmt.Sprintf("%f", value.(float64))
					case "map[string]interface {}":
						mm := value.(map[string]interface{})
						visitInfo(fmt.Sprintf("%s.%s", path, key), mm)
					default:
						mm := value.(map[string]interface{})
						visitInfo(fmt.Sprintf("%s.%s", path, key), mm)
					}
				}
			}
			equip := mapp["Equip"].(map[string]interface{})
			visitInfo("Equip", equip)

			removeTag := func(str string) string {
				for strings.Contains(str, "<") {
					i := strings.Index(str, "<")
					j := strings.Index(str[i:], ">")
					if j >= 0 {
						str = str[:i] + str[i+j+1:]
					} else {
						break
					}
				}
				return str
			}

			stoneStrings := make([][]string, 0)
			neckStrings := make([][]string, 0)
			earStrings := make([][]string, 0)
			ringStrings := make([][]string, 0)
			characterClass := ""

			for key, value := range propertyList {
				if strings.Contains(value, "전용") {
					value = removeTag(value)
					value = strings.ReplaceAll(value, "전용", "")
					value = strings.Trim(value, " ")
					if characterClass == "" {
						characterClass = value
						log.Printf("CLASS: %s\n", characterClass)
					}
				}
				if strings.HasSuffix(key, "Element_005.value.Element_000") {
					if strings.Contains(value, "무작위 각인 효과") {
						//Element_000.value
						nameKey := strings.ReplaceAll(key, "Element_005.value.Element_000", "Element_000.value")
						nameString := propertyList[nameKey]
						nameString = removeTag(nameString)

						buffKey := strings.ReplaceAll(key, "Element_005.value.Element_000", "Element_005.value.Element_001")
						buffString := propertyList[buffKey]
						buffString = strings.ReplaceAll(buffString, "<BR>", ";")
						buffString = strings.ReplaceAll(buffString, "활성도", "Lv")
						buffString = removeTag(buffString)

						log.Printf("ITEM %s %s", nameString, buffString)

						if strings.Contains(nameString, "의 돌") {
							stoneStrings = append(stoneStrings, []string{"[내꺼]" + nameString, buffString, "0", "-"})
						}
					}
				}
				if strings.HasSuffix(key, "Element_007.value.Element_000") {
					if strings.Contains(value, "무작위 각인 효과") {
						//Element_000.value
						nameKey := strings.ReplaceAll(key, "Element_007.value.Element_000", "Element_000.value")
						nameString := propertyList[nameKey]
						nameString = removeTag(nameString)

						qualityKey := strings.ReplaceAll(key, "Element_007.value.Element_000", "Element_001.value.qualityValue")
						qualityString := propertyList[qualityKey]
						if strings.HasPrefix(qualityString, "-1.") {
							qualityString = "-"
						} else {
							qualityString = fmt.Sprintf("%d", parseInt(qualityString))
						}

						buffKey := strings.ReplaceAll(key, "Element_007.value.Element_000", "Element_007.value.Element_001")
						buffString := propertyList[buffKey]
						buffString = strings.ReplaceAll(buffString, "<BR>", ";")
						buffString = strings.ReplaceAll(buffString, "활성도", "Lv")
						buffString = removeTag(buffString)

						statKey := strings.ReplaceAll(key, "Element_007.value.Element_000", "Element_006.value.Element_001")
						if statString, ok := propertyList[statKey]; ok {
							statStrings := strings.Split(statString, "<BR>")
							for _, statString1 := range statStrings {
								plus := strings.Index(statString1, "+")
								statString1 = fmt.Sprintf("[%s] %s", strings.Trim(statString1[:plus], " "), statString1[plus:])
								buffString += ";" + statString1
							}
						}
						log.Printf("ITEM %s %s", nameString, buffString)

						if strings.Contains(nameString, "목걸이") {
							neckStrings = append(neckStrings, []string{"[내꺼]" + nameString, buffString, "0", qualityString})
						} else if strings.Contains(nameString, "귀걸이") {
							earStrings = append(earStrings, []string{"[내꺼]" + nameString, buffString, "0", qualityString})
						} else if strings.Contains(nameString, "반지") {
							ringStrings = append(ringStrings, []string{"[내꺼]" + nameString, buffString, "0", qualityString})
						}
					}
				}
			}
			return characterClass, [][][]string{
				stoneStrings, neckStrings, earStrings, ringStrings,
			}
		}
	}
}

func loginStove() {
	driver := getWebDriver()
	for {
		driver.SetImplicitWaitTimeout(10 * time.Second)
		err := driver.Get("https://lostark.game.onstove.com/Main")
		panicIfError(err)

		sleepShortly()
		success, err := driver.ExecuteScript(`
			var login = document.querySelectorAll("#login-btn");
			if (login.length == 0) {
				return "false";
			}
			if (login[0].click) {
				login[0].click();
				return "true";
			} else {
				return "retry";
			}
		`, nil)
		panicIfError(err)
		if success == "false" {
			break
		} else if success == "retry" {
			sleepShortly()
			continue
		}

		for {
			time.Sleep(time.Second * 2)
			status, err := driver.Status()
			panicIfError(err)
			if !status.Ready {
				continue
			}
			if url, err := driver.CurrentURL(); err == nil {
				if strings.HasPrefix(url, "https://member.onstove.com/auth/") {
					continue
				}
			}
			return
		}
	}
}

func openAuction() {
	driver := getWebDriver()
	err := driver.Get("https://lostark.game.onstove.com/Auction")
	panicIfError(err)
	for {
		sleepShortly()
		detailBtn, err := driver.FindElement(selenium.ByCSSSelector, ".button--deal-detail")
		if err != nil {
			continue
		}
		if detailBtn != nil {
			detailBtn.Click()
			break
		}
	}
}

func selectDetailOption(className string, optionName string) {
	driver := getWebDriver()
	for {
		sleepShortly()
		optionGroup, err := driver.FindElement(selenium.ByCSSSelector, className)
		if err != nil || optionGroup == nil {
			continue
		}
		title, err := optionGroup.FindElement(selenium.ByCSSSelector, ".lui-select__title")
		if err != nil || title == nil {
			continue
		}
		optionGroup.Click()
		sleepShortly()
		option, err := optionGroup.FindElement(selenium.ByCSSSelector, ".lui-select__option")
		if err != nil || option == nil {
			continue
		}
		options, err := option.FindElements(selenium.ByTagName, "label")
		if err != nil || options == nil {
			continue
		}
		for _, opt := range options {
			itext, err := opt.GetAttribute("innerText")
			if err != nil {
				continue
			}
			if itext == optionName {
				opt.Click()
				sleepShortly()
				return
			}
		}
	}
}

func selectEtcDetailOption(idName string, optionName string) {
	driver := getWebDriver()
	for {
		sleepShortly()
		category, err := driver.FindElement(selenium.ByCSSSelector, idName)
		if err != nil || category == nil {
			continue
		}
		title, err := category.FindElement(selenium.ByCSSSelector, ".lui-select__title")
		if err != nil || title == nil {
			continue
		}
		title.Click()
		sleepShortly()
		option, err := category.FindElement(selenium.ByCSSSelector, ".lui-select__option")
		if err != nil || option == nil {
			continue
		}
		options, err := option.FindElements(selenium.ByTagName, "label")
		if err != nil || options == nil {
			continue
		}
		for _, opt := range options {
			itext, err := opt.GetAttribute("innerText")
			if err != nil {
				continue
			}
			if itext == optionName {
				opt.Click()
				sleepShortly()
				return
			}
		}
	}
}

func searchAndGetResults() ([][]string, bool) {
	driver := getWebDriver()
	for {
		sleepShortly()
		search, err := driver.FindElement(selenium.ByCSSSelector, ".lui-modal__search")
		if err != nil || search == nil {
			continue
		}
		search.Click()
		break
	}

	foundCount := 0
	page := 1
	retList := make([][]string, 0)
	for {
		time.Sleep(500 * time.Millisecond)
		// sleepShortly()

		// empty, err := driver.FindElement(selenium.ByCSSSelector, "#auctionListTbody .empty")
		// if err == nil || empty != nil {
		// 	emptyText, _ := empty.GetAttribute("innerText")
		// 	if strings.Trim(emptyText, " ") == "경매장 연속 검색으로 인해 검색 이용이 최대 5분간 제한되었습니다." {
		// 		log.Println("이용 제한으로 5분 대기")
		// 		time.Sleep(5 * time.Minute)
		// 		driver.ExecuteScript(fmt.Sprintf("paging.page(%d);", page), nil)
		// 		continue
		// 	}
		// 	return retList, false
		// }

		for {
			sleepShortly()

			success, err := driver.ExecuteScript(`
				var listBody = document.querySelectorAll("#auctionListTbody .empty");
				if (listBody.length == 0) {
					return "false";
				}
				if (listBody[0].innerText.trim() == "경매장 연속 검색으로 인해 검색 이용이 최대 5분간 제한되었습니다.") {
					return "5min";
				}
				return "true";
			`, nil)
			panicIfError(err)

			switch success {
			case "true":
				return retList, false
			case "5min":
				log.Println("이용 제한으로 1분 대기")
				time.Sleep(time.Minute)
				driver.ExecuteScript(fmt.Sprintf("paging.page(%d);", page), nil)
				continue
			}
			break
		}

		if page == 1 {
			for {
				sleepShortly()

				success, err := driver.ExecuteScript(`
					var priceButton = document.querySelectorAll("#BUY_PRICE");
					if (priceButton.length == 0) {
						return "false";
					}
					var sort = priceButton[0].parentElement.getAttribute("aria-sort");
					if (sort != "ascending") {
						priceButton[0].click();
					}
					return "true";
				`, nil)
				panicIfError(err)
				if err != nil || success.(string) != "true" {
					time.Sleep(500 * time.Millisecond)
					continue
				}
				break
			}
		}

		auctionList, err := driver.FindElement(selenium.ByCSSSelector, "#auctionListTbody")
		if err != nil || auctionList == nil {
			continue
		}

		// table 내용을 한번에 가져온다.
		itemListScripts := `
			var ss = '';
			var trs = document.querySelectorAll("#auctionListTbody tr");
			for (var i = 0; i < trs.length; i++) {
				var tr = trs[i];
				var name = tr.getElementsByClassName('name')[0].innerText;
				var effect = tr.getElementsByClassName('effect')[0].innerText;
				var price = tr.getElementsByClassName('price-buy')[0].innerText;
				var quality = '';
				if (tr.getElementsByClassName('quality').length > 0) {
					quality = tr.getElementsByClassName('quality')[0].innerText;
					quality = quality.trim();
				}
				effect = effect.replaceAll("\n", ';');
				price = price.replaceAll(',', '').trim();
				ss += name + ',' + effect + ',' + price;
				ss += ',' + quality;
				ss += "\n";
			}
			return ss;
		`
		ret, err := driver.ExecuteScript(itemListScripts, nil)
		if err != nil {
			continue
		}
		lines := strings.Split(ret.(string), "\n")

		for _, line := range lines {
			part := strings.Split(line, ",")
			if len(part) < 3 {
				continue
			}
			if strings.Trim(part[2], " ") != "-" {
				foundCount++
				retList = append(retList, part)
			}
		}

		if foundCount < ctx.AuctionItemCount {
			page++
			log.Printf("%d 페이지 추가 검색\n", page)
			driver.ExecuteScript(fmt.Sprintf("paging.page(%d);", page), nil)
		} else {
			return retList, false
		}
	}
}

func selectSkillMinLevel(idName string, level string) {
	driver := getWebDriver()
	for {
		sleepShortly()
		input, err := driver.FindElement(selenium.ByCSSSelector, idName)
		if err != nil || input == nil {
			continue
		}
		input.SendKeys(level)
		break
	}
}
