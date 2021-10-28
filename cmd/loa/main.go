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
	"syscall"

	"github.com/gorilla/websocket"
	"github.com/syspro86/lostark-auction-finder/pkg/loa"
	"github.com/tebeka/selenium"
	"github.com/tebeka/selenium/chrome"
	"github.com/zserge/lorca"
)

func getWebDriver() selenium.WebDriver {
	caps := selenium.Capabilities{"browserName": "chrome"}
	caps.AddChrome(chrome.Capabilities{Args: []string{fmt.Sprintf("--user-data-dir=%s", toolConfig.ChromeUserDataPath)}})
	driver, err := selenium.NewRemote(caps, toolConfig.SeleniumURL)
	printIfError(err)
	return driver
}

//go:embed web
var web embed.FS

func main() {
	toolConfig.Load("tool.json")

	if toolConfig.SeleniumURL == "" {
		toolConfig.SeleniumURL = "http://localhost:4444/wd/hub"
	}
	if toolConfig.ChromeUserDataPath == "" {
		cmd := exec.Command("chromedriver.exe", "--port=4444", "--url-base=wd/hub", "--verbose")
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		cmd.Start()
		defer cmd.Process.Kill()

		// service, err := selenium.NewChromeDriverService("chromedriver.exe", 4444)
		// panicIfError(err)
		// defer service.Stop()

		user, err := user.Current()
		panicIfError(err)
		toolConfig.ChromeUserDataPath = fmt.Sprintf("%s\\AppData\\Local\\Google\\Chrome\\User Data\\", user.HomeDir)
	}

	var ctx = Context{
		CharacterName:      "",
		LearnedBuffs:       map[string]int{},
		SupposedStoneLevel: []int{6, 6, 3},
		Grade:              "유물",
		AuctionItemCount:   10,
		TargetBuffs:        map[string]int{},
		TargetQuality:      "전체 품질",
		MaxDebuffLevel:     1,
	}
	ctx.Load(toolConfig.FileBase + "config.json")

	if runtime.GOOS == "windows" {
		upgrader := websocket.Upgrader{
			ReadBufferSize:  1024 * 1024,
			WriteBufferSize: 1024 * 1024,
		}

		startSearch := func(conn *websocket.Conn) {
			_, data, _ := conn.ReadMessage()
			if err := json.Unmarshal(data, &ctx); err != nil {
				return
			}
			if ctx.CharacterName == "" {
				return
			}
			ctx.Save(toolConfig.FileBase + "config.json")

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
				job := TripodJob{
					Web:       WebClient{Driver: driver},
					LogWriter: writeFunction,
					Ctx:       ctx,
				}
				job.Start()
			} else {
				job := AccessoryJob{
					Web:       WebClient{Driver: driver},
					LogWriter: writeFunction,
					Ctx:       ctx,
				}
				job.Start()
			}
		}
		webSub, _ := fs.Sub(web, "web")
		http.Handle("/", http.FileServer(http.FS(webSub)))
		http.HandleFunc("/loa/const", func(resp http.ResponseWriter, req *http.Request) {
			resp.Header().Add("Content-Type", "application/json")
			data, _ := json.Marshal(loa.Const)
			resp.Write(data)
		})
		http.HandleFunc("/context", func(resp http.ResponseWriter, req *http.Request) {
			resp.Header().Add("Content-Type", "application/json")
			data, _ := json.Marshal(ctx)
			resp.Write(data)
		})
		http.HandleFunc("/start", func(resp http.ResponseWriter, req *http.Request) {
			conn, err := upgrader.Upgrade(resp, req, nil)
			if err != nil {
				log.Printf("upgrader.Upgrade: %v", err)
				return
			}
			defer conn.Close()

			startSearch(conn)
		})
		// if err := exec.Command("rundll32", "url.dll,FileProtocolHandler", "http://localhost:5555/").Start(); err != nil {
		// 	panic(err)
		// }

		httpStop := make(chan bool)
		go func() {
			http.ListenAndServe(":5555", nil)
			httpStop <- true
		}()

		ui, err := lorca.New("http://localhost:5555", "", 480, 320)
		if err != nil {
			log.Fatal(err)
		}
		defer ui.Close()

		// ui.Load("http://localhost:5555")

		select {
		case <-ui.Done():
		case <-httpStop:
		}

		// if err := http.ListenAndServe(":5555", nil); err != nil {
		// 	panicIfError(err)
		// }

		// wv := webview.New(false)
		// defer wv.Destroy()

		// wv.SetTitle("LostArk Auction Finder")
		// wv.SetSize(800, 600, webview.HintNone)
		// wv.Navigate("http://localhost:5555/")
		// wv.Run()
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
			job := TripodJob{
				Web:       WebClient{Driver: driver},
				LogWriter: writeFunction,
				Ctx:       ctx,
			}
			job.Start()
		} else {
			job := AccessoryJob{
				Web:       WebClient{Driver: driver},
				LogWriter: writeFunction,
				Ctx:       ctx,
			}
			job.Start()
		}
	}
}
