package tools

import (
	"encoding/json"
	"log"
	"os"
)

type ToolConfig struct {
	FileBase           string
	SeleniumURL        string
	ChromeUserDataPath string
	CacheSearchResult  bool
	CachePath          string
}

var Config = ToolConfig{
	CacheSearchResult: true,
	CachePath:         "cache/",
}

func (cfg *ToolConfig) Load(fileName string) {
	if data, err := os.ReadFile(fileName); err == nil {
		tmp := ToolConfig{}
		if err := json.Unmarshal(data, &tmp); err == nil {
			*cfg = tmp
		} else {
			log.Println(err)
		}
	}
}
