package log

import (
	"encoding/json"
	"os"
	"testing"
)

func TestGetLevel(t *testing.T) {
	for _, levelStr := range level {
		num := GetLevel(levelStr)
		if num < 0 || num > 7 {
			t.Fatalf("get log level failed: %d\n", num)
		}
	}

	num := GetLevel("default")
	if num != 7 {
		t.Errorf("defaultLogLevel set failed: %d\n", num)
	}

	initLogLevel("error")
	Debug("my book is bought in the year of ", 2016)
	Info("this %s cat is %v years old", "yellow", 3)
	Notice("Notice log")
	Warn("json is a type of kv like", map[string]int{"key": 2016})
	Error("error log")
	Critical("critical log")
	Alert("alert log")
	Emergency("emergency log")
}

func initLogLevel(level string) {
	fileName := "/tmp/test.log"

	logConf := struct {
		FileName string `json:"filename"`
		Level    int    `json:"level"`
	}{
		FileName: fileName,
		Level:    GetLevel(level),
	}

	configuration, err := json.Marshal(logConf)
	if err != nil {
		panic(err)
	}
	Init(string(configuration))
	os.Remove(fileName)
}
