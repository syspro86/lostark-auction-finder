package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os/user"
	"reflect"
	"strings"
	"time"

	"github.com/tebeka/selenium"
	"github.com/zserge/lorca"
)

type WebClient struct {
	UseSelenium bool
	Driver      selenium.WebDriver
	GUI         lorca.UI
	Closers     []CloseFunction
}

type CharacterInfo struct {
	CharacterName string
	ClassName     string
	Stone         [][]string
	Neck          [][]string
	Ear           [][]string
	Ring          [][]string
}

type WebClientLoadTimeout struct{}

func (err *WebClientLoadTimeout) Error() string {
	return "Load timeout"
}

func (client *WebClient) InitSelenium() {
	if toolConfig.SeleniumURL == "" {
		closeChrome := client.InitChromeDriver()
		client.Closers = append(client.Closers, closeChrome)
		toolConfig.SeleniumURL = "http://localhost:4444/wd/hub"
	}
	if toolConfig.ChromeUserDataPath == "" {
		user, err := user.Current()
		panicIfError(err)
		toolConfig.ChromeUserDataPath = fmt.Sprintf("%s\\AppData\\Local\\Google\\Chrome\\User Data\\", user.HomeDir)
	}
}

func (client *WebClient) Close() {
	for _, cl := range client.Closers {
		cl()
	}
	client.Closers = []CloseFunction{}
}

func (client *WebClient) GetItemsFromCharacter(characterName string) (CharacterInfo, error) {
	charInfo := CharacterInfo{}
	for {
		profileJson := ""
		if client.UseSelenium {
			client.Driver.SetImplicitWaitTimeout(10 * time.Second)
			err := client.Driver.Get(fmt.Sprintf("https://lostark.game.onstove.com/Profile/Character/%s", characterName))
			panicIfError(err)

			sleepShortly()
			success, err := client.Driver.ExecuteScript(`
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
			profileJson = success.(string)
		} else {
			ui, err := lorca.New("", "", 400, 400, "--headless")
			panicIfError(err)
			defer ui.Close()

			ui.Load(fmt.Sprintf("https://lostark.game.onstove.com/Profile/Character/%s", characterName))
			val := ui.Eval(`document.querySelectorAll("#profile-ability script").length`)
			for cnt := 0; cnt < 10 && val.Int() == 0; cnt++ {
				time.Sleep(time.Second)
				val = ui.Eval(`document.querySelectorAll("#profile-ability script").length`)
			}
			if val.Int() == 0 {
				return CharacterInfo{}, &WebClientLoadTimeout{}
			}
			val = ui.Eval(`
				document.querySelectorAll("#profile-ability script")[0].innerText
			`)
			profileJson = val.String()
		}

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
			for key, value := range propertyList {
				if strings.Contains(value, "전용") {
					value = removeTag(value)
					value = strings.ReplaceAll(value, "전용", "")
					value = strings.Trim(value, " ")
					if charInfo.ClassName == "" {
						charInfo.ClassName = value
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
						// log.Printf("ITEM %s %s", nameString, buffString)

						if strings.Contains(nameString, "의 돌") {
							charInfo.Stone = append(charInfo.Stone, []string{"[내꺼]" + nameString, buffString, "0", "-"})
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
						// log.Printf("ITEM %s %s", nameString, buffString)

						if strings.Contains(nameString, "목걸이") {
							charInfo.Neck = append(charInfo.Neck, []string{"[내꺼]" + nameString, buffString, "0", qualityString})
						} else if strings.Contains(nameString, "귀걸이") {
							charInfo.Ear = append(charInfo.Ear, []string{"[내꺼]" + nameString, buffString, "0", qualityString})
						} else if strings.Contains(nameString, "반지") {
							charInfo.Ring = append(charInfo.Ring, []string{"[내꺼]" + nameString, buffString, "0", qualityString})
						}
					}
				}
			}
			return charInfo, nil
		}
	}
}

func (client *WebClient) loginStove() {
	for {
		client.Driver.SetImplicitWaitTimeout(10 * time.Second)
		err := client.Driver.Get("https://lostark.game.onstove.com/Main")
		panicIfError(err)

		sleepShortly()
		success, err := client.Driver.ExecuteScript(`
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
			status, err := client.Driver.Status()
			panicIfError(err)
			if !status.Ready {
				continue
			}
			if url, err := client.Driver.CurrentURL(); err == nil {
				if strings.HasPrefix(url, "https://member.onstove.com/auth/") {
					continue
				}
			}
			return
		}
	}
}

func (client *WebClient) openAuction() {
	err := client.Driver.Get("https://lostark.game.onstove.com/Auction")
	panicIfError(err)
	for {
		sleepShortly()
		detailBtn, err := client.Driver.FindElement(selenium.ByCSSSelector, ".button--deal-detail")
		if err != nil {
			continue
		}
		if detailBtn != nil {
			detailBtn.Click()
			break
		}
	}
}

func (client *WebClient) selectDetailOption(className string, optionName string) {
	for {
		sleepShortly()
		optionGroup, err := client.Driver.FindElement(selenium.ByCSSSelector, className)
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

func (client *WebClient) selectEtcDetailOption(idName string, optionName string) {
	for {
		sleepShortly()
		category, err := client.Driver.FindElement(selenium.ByCSSSelector, idName)
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

func (client *WebClient) selectEtcDetailText(idName string, keys string) {
	for {
		sleepShortly()
		text, err := client.Driver.FindElement(selenium.ByCSSSelector, idName)
		if err != nil || text == nil {
			continue
		}
		text.Click()
		text.SendKeys(keys)
		return
	}
}

func (client *WebClient) searchAndGetResults(itemCount int) ([][]string, bool) {
	for {
		sleepShortly()
		search, err := client.Driver.FindElement(selenium.ByCSSSelector, ".lui-modal__search")
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

		// empty, err := client.Driver.FindElement(selenium.ByCSSSelector, "#auctionListTbody .empty")
		// if err == nil || empty != nil {
		// 	emptyText, _ := empty.GetAttribute("innerText")
		// 	if strings.Trim(emptyText, " ") == "경매장 연속 검색으로 인해 검색 이용이 최대 5분간 제한되었습니다." {
		// 		log.Println("이용 제한으로 5분 대기")
		// 		time.Sleep(5 * time.Minute)
		// 		client.Driver.ExecuteScript(fmt.Sprintf("paging.page(%d);", page), nil)
		// 		continue
		// 	}
		// 	return retList, false
		// }

		for {
			sleepShortly()

			success, err := client.Driver.ExecuteScript(`
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
				client.Driver.ExecuteScript(fmt.Sprintf("paging.page(%d);", page), nil)
				continue
			}
			break
		}

		if page == 1 {
			for {
				sleepShortly()

				success, err := client.Driver.ExecuteScript(`
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

		auctionList, err := client.Driver.FindElement(selenium.ByCSSSelector, "#auctionListTbody")
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
		ret, err := client.Driver.ExecuteScript(itemListScripts, nil)
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

		if foundCount < itemCount {
			page++
			log.Printf("%d 페이지 추가 검색\n", page)
			client.Driver.ExecuteScript(fmt.Sprintf("paging.page(%d);", page), nil)
		} else {
			return retList, false
		}
	}
}

func (client *WebClient) selectSkillMinLevel(idName string, level string) {
	for {
		sleepShortly()
		input, err := client.Driver.FindElement(selenium.ByCSSSelector, idName)
		if err != nil || input == nil {
			continue
		}
		input.SendKeys(level)
		break
	}
}
