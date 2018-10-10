package lreindex

import (
	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/persist"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	args := make([]string, 0)
	conf.Cfg = conf.InitConfig(args)
	persist.InitPersistGlobal()
	chain.InitGlobalChain()
	log.Init()
	os.Exit(m.Run())
}

//func TestReindex(t *testing.T) {
//	Reindex()
//}
