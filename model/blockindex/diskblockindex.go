package blockindex

import (

	"io"

	"github.com/btcboost/copernicus/util"
	"github.com/btcboost/copernicus/persist/db"
)




func (bIndex *BlockIndex)GetSerializeList()[]string{
	dump_list := []string{"Height","Status", "TxCount", "File", "DataPos","UndoPos","Header"}
	return dump_list
}

func (bIndex *BlockIndex) Serialize(w io.Writer) error {
	clientVersion := 160000
	err := util.WriteVarLenInt(w, uint64(clientVersion))
	if err != nil {
		return err
	}
	return db.SerializeOP(w, bIndex)
}

