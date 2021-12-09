package main

import (
	"os/exec"
	"syscall"
)

func (client *WebClient) InitChromeDriver() CloseFunction {
	cmd := exec.Command("chromedriver.exe", "--port=4444", "--url-base=wd/hub", "--verbose")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	cmd.Start()
	return func() {
		cmd.Process.Kill()
	}
}
