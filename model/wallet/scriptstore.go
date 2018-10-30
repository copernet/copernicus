package wallet

import (
	"sync"

	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/util"
)

type ScriptStore struct {
	lock    *sync.RWMutex
	scripts map[string]*script.Script
}

func NewScriptStore() *ScriptStore {
	return &ScriptStore{
		lock:    new(sync.RWMutex),
		scripts: make(map[string]*script.Script),
	}
}

func (ss *ScriptStore) AddScript(s *script.Script) {
	scriptHash := util.Hash160(s.Bytes())

	ss.lock.Lock()
	defer ss.lock.Unlock()

	ss.scripts[string(scriptHash)] = s
}

func (ss *ScriptStore) GetScript(scriptHash []byte) *script.Script {
	ss.lock.RLock()
	defer ss.lock.RUnlock()

	if s, ok := ss.scripts[string(scriptHash)]; ok {
		return s
	}
	return nil
}
