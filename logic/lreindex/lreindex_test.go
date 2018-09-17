package lreindex

import (
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/persist/global"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	global.InitPersistGlobal()
	chain.InitGlobalChain()
	log.Init()
	os.Exit(m.Run())
}

func TestReindex(t *testing.T) {
	Reindex()
}
