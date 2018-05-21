package blockindex

import (

	"io"

	"github.com/btcboost/copernicus/util"
	"github.com/btcboost/copernicus/persist/db"
)




func (bIndex *BlockIndex)GetSerializeList()[]string{
	dumpList := []string{"Height","Status", "TxCount", "File", "DataPos","UndoPos","Header"}
	return dumpList
}

func (bIndex *BlockIndex) Serialize(w io.Writer) error {
	clientVersion := 160000
	err := util.WriteVarLenInt(w, uint64(clientVersion))
	if err != nil {
		return err
	}
	return db.SerializeOP(w, bIndex)
}

func (bIndex *BlockIndex) Unserialize(r io.Reader) error {
	_, err := util.ReadVarLenInt(r)
	if err != nil {
		return err
	}

	return db.UnserializeOP(r, bIndex)
}