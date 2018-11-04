package wallet

import (
	"sync"

	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/util"
)

type ScriptStore struct {
	sync.RWMutex
	scripts map[string]*script.Script
}

func NewScriptStore() *ScriptStore {
	return &ScriptStore{
		scripts: make(map[string]*script.Script),
	}
}

func (ss *ScriptStore) AddScript(s *script.Script) {
	scriptHash := util.Hash160(s.Bytes())

	ss.Lock()
	defer ss.Unlock()

	ss.scripts[string(scriptHash)] = s
}

func (ss *ScriptStore) GetScript(scriptHash []byte) *script.Script {
	ss.RLock()
	defer ss.RUnlock()

	if s, ok := ss.scripts[string(scriptHash)]; ok {
		return s
	}
	return nil
}
