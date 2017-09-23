package utils

import (
	"strconv"
	"strings"
	"sync"
)

var lock sync.Mutex
var MapArgs map[string]string
var MapMultiArgs map[string][]string

func ParseParameters(argc int, argv []string) {
	lock.Lock()
	defer lock.Unlock()

	MapArgs = make(map[string]string)
	MapMultiArgs = make(map[string][]string)

	for i := 0; i < argc; i++ {
		str := argv[i]
		var strValue string
		isIndex := strings.IndexByte(str, '=')
		if isIndex != -1 {
			strValue = str[isIndex+1:]
			str = str[:isIndex]
		}

		if str[0] != '-' {
			break
		}
		// Interpret --foo as -foo.
		// If both --foo and -foo are set, the last takes effect.
		if len(str) > 1 && str[1] == '-' {
			str = str[1:]
		}

		InterpretNegativeSetting(&str, &strValue)
		MapArgs[str] = strValue
		var tmpSlice []string
		if v, ok := MapMultiArgs[str]; ok {
			tmpSlice = v
		} else {
			tmpSlice = make([]string, 0)
		}
		tmpSlice = append(tmpSlice, strValue)
		MapMultiArgs[str] = tmpSlice

	}

}

func InterpretBool(valueStr string) bool {
	if len(valueStr) == 0 {
		return true
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		panic(err)
	}
	if value != 0 {
		return true
	}
	return false
}

func InterpretNegativeSetting(keySTr, valueStr *string) {
	if len(*keySTr) > 3 && (*keySTr)[0] == '-' && (*keySTr)[1] == 'n' && (*keySTr)[2] == 'o' {
		*keySTr = "-" + (*keySTr)[3:]
		if InterpretBool(*valueStr) {
			*valueStr = "0"
		} else {
			*valueStr = "1"
		}
	}
}

func GetArg(strArg string, deFault int64) int64 {
	lock.Lock()
	defer lock.Unlock()
	if v, ok := MapArgs[strArg]; ok {
		tmpV, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			panic(err)
		} else {
			return tmpV
		}
	}

	return deFault
}
