package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os/exec"
	"os/user"
	"runtime"

	"github.com/gorilla/websocket"
	"github.com/syspro86/lostark-auction-finder/pkg/tools"
	"github.com/tebeka/selenium"
	"github.com/tebeka/selenium/chrome"
)

var ctx = tools.Context{
	CharacterName:      "",
	LearnedBuffs:       map[string]int{},
	SupposedStoneLevel: []int{6, 6, 3},
	Grade:              "유물",
	AuctionItemCount:   10,
	Budget:             10_000,
	TargetBuffs:        map[string]int{},
	TargetQuality:      "전체 품질",
	MaxDebuffLevel:     1,
}

var _driver selenium.WebDriver

func getWebDriver() selenium.WebDriver {
	if _driver == nil {
		caps := selenium.Capabilities{"browserName": "chrome"}
		caps.AddChrome(chrome.Capabilities{Args: []string{fmt.Sprintf("--user-data-dir=%s", tools.Config.ChromeUserDataPath)}})
		driver, err := selenium.NewRemote(caps, tools.Config.SeleniumURL)
		panicIfError(err)
		_driver = driver
	}
	return _driver
}

//go:embed web
var web embed.FS

func main() {
	tools.Config.Load("tool.json")
	ctx.Load(tools.Config.FileBase + "config.json")

	if tools.Config.SeleniumURL == "" {
		tools.Config.SeleniumURL = "http://localhost:4444/wd/hub"
	}
	if tools.Config.ChromeUserDataPath == "" {
		service, err := selenium.NewChromeDriverService("chromedriver.exe", 4444)
		panicIfError(err)
		defer service.Stop()

		user, err := user.Current()
		panicIfError(err)
		tools.Config.ChromeUserDataPath = fmt.Sprintf("%s\\AppData\\Local\\Google\\Chrome\\User Data\\", user.HomeDir)
	}

	if runtime.GOOS == "windows" {
		upgrader := websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		}

		startSearch := func(conn *websocket.Conn) {
			_, data, _ := conn.ReadMessage()

			ctx.CharacterName = string(data)
			if ctx.CharacterName == "" {
				return
			}

			writeFunction := func(msgtype string, message interface{}) {
				data, _ := json.Marshal(map[string]interface{}{
					"type": msgtype,
					"data": message,
				})
				conn.WriteMessage(websocket.TextMessage, data)
			}
			writeFunction("log", fmt.Sprintf("캐릭터: %s", ctx.CharacterName))

			driver := getWebDriver()
			defer driver.Close()
			if len(ctx.TargetTripods) > 0 {
				suggestTripod()
			} else {
				suggestAccessory(writeFunction)
			}
		}
		webSub, _ := fs.Sub(web, "web")
		http.Handle("/", http.FileServer(http.FS(webSub)))
		http.HandleFunc("/start", func(resp http.ResponseWriter, req *http.Request) {
			conn, err := upgrader.Upgrade(resp, req, nil)
			if err != nil {
				log.Printf("upgrader.Upgrade: %v", err)
				return
			}
			defer conn.Close()

			startSearch(conn)
		})
		if err := exec.Command("rundll32", "url.dll,FileProtocolHandler", "http://localhost:5555/").Start(); err != nil {
			panic(err)
		}
		if err := http.ListenAndServe(":5555", nil); err != nil {
			panicIfError(err)
		}
	} else {
		log.Printf("캐릭터: %s\n", ctx.CharacterName)
		if ctx.CharacterName == "" {
			return
		}

		writeFunction := func(msgtype string, message interface{}) {
			log.Println(message)
		}

		driver := getWebDriver()
		defer driver.Close()
		if len(ctx.TargetTripods) > 0 {
			suggestTripod()
		} else {
			suggestAccessory(writeFunction)
		}
	}
}
