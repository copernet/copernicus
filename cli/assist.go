package main

import (
	"flag"

	"fmt"

	"github.com/btcboost/copernicus/utils"
)

func main() {
	str := flag.String("str", "", "Input you original string")
	split := flag.String("split", "", "Split char")
	flag.Parse()
	result, err := utils.SplitHex(*str, *split)
	if err != nil {
		fmt.Println("err:" + err.Error())
	}
	fmt.Println(result)

}
