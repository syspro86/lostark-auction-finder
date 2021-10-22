package main

import (
	"strconv"
	"time"
)

func panicIfError(err error) {
	if err != nil {
		panic(err)
	}
}

func sleepShortly() {
	time.Sleep(time.Millisecond * 10)
}

func parseInt(s string) int {
	i, _ := strconv.ParseInt(s, 10, 32)
	return int(i)
}

func arrayIndexOf(array []string, val string) int {
	for i, v := range array {
		if v == val {
			return i
		}
	}
	return -1
}
