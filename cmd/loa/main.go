package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"runtime"

	log "github.com/sirupsen/logrus"

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

func initWebServer(client *WebClient) chan bool {
	type WSMessage struct {
		Type string
		Data Context
	}
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024 * 1024,
		WriteBufferSize: 1024 * 1024,
	}
	startSearch := func(conn *websocket.Conn) {
		defer conn.Close()

		writeFunction := func(msgtype string, message interface{}) {
			data, _ := json.Marshal(map[string]interface{}{
				"type": msgtype,
				"data": message,
			})
			conn.WriteMessage(websocket.TextMessage, data)
		}
		writeFunction("const", loa.Const)
		if runtime.GOOS == "windows" {
			var ctx = Context{}
			ctx.Load(toolConfig.FileBase + "config.json")
			writeFunction("context", ctx)

			client.Driver = getWebDriver()
			defer client.Driver.Close()
		}

		for {
			_, data, _ := conn.ReadMessage()
			log.Trace("New message from user")
			msg := WSMessage{}
			if err := json.Unmarshal(data, &msg); err != nil {
				return
			}
			log.Trace(msg)
			if msg.Type == "character" {
				ci, err := client.GetItemsFromCharacter(msg.Data.CharacterName)
				printIfError(err)
				if err == nil {
					writeFunction("character", ci)
				}
			} else if msg.Type == "search" {
				ctx := msg.Data
				if ctx.CharacterName == "" {
					return
				}
				if runtime.GOOS == "windows" {
					ctx.Save(toolConfig.FileBase + "config.json")
				}
				writeFunction("log", fmt.Sprintf("캐릭터: %s", ctx.CharacterName))

				if len(ctx.TargetTripods) > 0 {
					job := TripodJob{
						Web:       client,
						LogWriter: writeFunction,
						Ctx:       ctx,
					}
					job.Start()
				} else {
					job := AccessoryJob{
						Web:       client,
						LogWriter: writeFunction,
						Ctx:       ctx,
					}
					job.Start()
				}
			}
		}
	}
	webSub, _ := fs.Sub(web, "web")
	if runtime.GOOS != "windows" {
		webSub = os.DirFS("web")
	}
	http.Handle("/", http.FileServer(http.FS(webSub)))
	http.HandleFunc("/ws", func(resp http.ResponseWriter, req *http.Request) {
		conn, err := upgrader.Upgrade(resp, req, nil)
		if err != nil {
			log.Printf("upgrader.Upgrade: %v", err)
			return
		}
		go startSearch(conn)
	})

	httpStop := make(chan bool)
	go func() {
		http.ListenAndServe(":5555", nil)
		httpStop <- true
	}()
	return httpStop
}

func initUI() (func(), <-chan struct{}) {
	if runtime.GOOS == "windows" {
		ui, err := lorca.New("http://localhost:5555", "", 1000, 800)
		if err != nil {
			log.Fatal(err)
		}
		return func() { ui.Close() }, ui.Done()
	} else {
		return func() {}, make(chan struct{})
	}
}

func main() {
	toolConfig.Load("tool.json")
	if level, err := log.ParseLevel(toolConfig.LogLevel); err == nil {
		log.SetLevel(level)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	wc := &WebClient{}
	wc.InitSelenium()
	defer wc.Close()

	httpStop := initWebServer(wc)
	uiClose, uiStop := initUI()

	defer uiClose()
	select {
	case <-uiStop:
	case <-httpStop:
	}
}
