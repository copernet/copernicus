package blockindex

import (

	"io"

	"github.com/btcboost/copernicus/util"
	"bytes"
)




func (bIndex *BlockIndex)GetSerializeList()[]string{
	dumpList := []string{"Height","Status", "TxCount", "File", "DataPos","UndoPos","Header"}
	return dumpList
}

func (bIndex *BlockIndex) Serialize(w io.Writer) error {
	buf := bytes.NewBuffer(nil)
	clientVersion := 160000
	err := util.WriteVarLenInt(buf, uint64(clientVersion))
	if err != nil {
		return err
	}
	err = util.WriteElements(buf, bIndex.Height, bIndex.Status, bIndex.TxCount, bIndex.File, bIndex.DataPos, bIndex.UndoPos)
	if err != nil {
		return err
	}
	err = bIndex.Header.Serialize(buf)
	if err != nil {
		return err
	}

	dataLen := buf.Len()
	util.WriteVarLenInt(w, uint64(dataLen))
	_, err = w.Write(buf.Bytes())
	return err
}

func (bIndex *BlockIndex) Unserialize(r io.Reader) error {
	_, err := util.ReadVarLenInt(r)
	if err != nil {
		return err
	}
	err = util.ReadElements(r, &bIndex.Height, &bIndex.Status, &bIndex.TxCount, &bIndex.File, &bIndex.DataPos, &bIndex.UndoPos)
	if err != nil {
		return err
	}
	err = bIndex.Header.Unserialize(r)
	return err
}