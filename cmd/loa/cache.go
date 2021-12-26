package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func getCachePath(searchKey string) string {
	return fmt.Sprintf("%s%s.json", toolConfig.CachePath, searchKey)
}

func saveCacheResult(searchKey string, result [][]string) {
	saveCacheResultFile(searchKey, result)
	// saveCacheResultDB(searchKey+".json", result)
}

func saveCacheResultFile(searchKey string, result [][]string) {
	data, _ := json.MarshalIndent(result, "", "  ")
	if _, err := os.Stat(toolConfig.CachePath); errors.Is(err, os.ErrNotExist) {
		os.Mkdir(toolConfig.CachePath, 0755)
	}
	os.WriteFile(getCachePath(searchKey), data, 0644)
}

func loadCacheResult(searchKey string) [][]string {
	return loadCacheResultFile(searchKey)
	// return loadCacheResultDB(searchKey + ".json")
}

func loadCacheResultFile(searchKey string) [][]string {
	// 캐시 데이터가 한시간 이상 지났으면 삭제
	if stat, err := os.Stat(getCachePath(searchKey)); err == nil {
		if stat.ModTime().Add(time.Hour * 24).Before(time.Now()) {
			os.Remove(toolConfig.FileBase + searchKey)
		}
	}
	if data, err := os.ReadFile(getCachePath(searchKey)); err == nil {
		tmp := new([][]string)
		json.Unmarshal(data, tmp)
		return *tmp
	}
	return [][]string{}
}
