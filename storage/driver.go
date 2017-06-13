package storage

import (
	"github.com/siddontang/ledisdb/store/driver"
)

func ListDrivers() []string {
	return driver.ListStores()
}
