package storage

import (
	"fmt"

	lediscfg "github.com/siddontang/ledisdb/config"
	"github.com/siddontang/ledisdb/ledis"
)

func init() {
	cfg := lediscfg.NewConfigDefault()
	l, err := ledis.Open(cfg)
	if err != nil {
		fmt.Println(err)
	}
	db, _ := l.Select(0)
	db.Set([]byte("test_key"), []byte("test_value"))
	res, err := db.Get([]byte("test_key"))
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(res))
	drivers := ListDrivers()
	for _, v := range drivers {
		fmt.Printf("driver %s is registered\n", v)
	}
}
