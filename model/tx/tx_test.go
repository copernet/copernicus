package tx

import (
	"bytes"
	"encoding/hex"
	"testing"

	"fmt"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/txin"
	"github.com/copernet/copernicus/model/txout"
	"github.com/copernet/copernicus/util"
)

var tests = []struct {
	GetHash string
	txRaw   string
	tx      *Tx
}{
	{
		GetHash: "8c14f0db3df150123e6f3dbbf30f8b955a8249b62ac1d1ff16284aefa3d06d87",
		txRaw:   "01000000010000000000000000000000000000000000000000000000000000000000000000ffffffff08044c86041b020602ffffffff0100f2052a010000004341041b0e8c2567c12536aa13357b79a073dc4444acb83c4ec7a0e2f99dd7457516c5817242da796924ca4e99947d087fedf9ce467cb9f7c6287078f801df276fdf84ac00000000",
		tx: &Tx{
			version: 1,
			ins: []*txin.TxIn{
				txin.NewTxIn(outpoint.NewOutPoint(util.Hash{}, 0xffffffff),
					script.NewScriptRaw([]byte{0x04, 0x4c, 0x86, 0x04, 0x1b, 0x02, 0x06, 0x02}),
					0xffffffff),
			},
			outs: []*txout.TxOut{
				txout.NewTxOut(0x12a05f200, script.NewScriptRaw([]byte{
					0x41, // OP_DATA_65
					0x04, 0x1b, 0x0e, 0x8c, 0x25, 0x67, 0xc1, 0x25,
					0x36, 0xaa, 0x13, 0x35, 0x7b, 0x79, 0xa0, 0x73,
					0xdc, 0x44, 0x44, 0xac, 0xb8, 0x3c, 0x4e, 0xc7,
					0xa0, 0xe2, 0xf9, 0x9d, 0xd7, 0x45, 0x75, 0x16,
					0xc5, 0x81, 0x72, 0x42, 0xda, 0x79, 0x69, 0x24,
					0xca, 0x4e, 0x99, 0x94, 0x7d, 0x08, 0x7f, 0xed,
					0xf9, 0xce, 0x46, 0x7c, 0xb9, 0xf7, 0xc6, 0x28,
					0x70, 0x78, 0xf8, 0x01, 0xdf, 0x27, 0x6f, 0xdf,
					0x84, // 65-byte signature
					0xac, // OP_CHECKSIG
				})),
			},
		},
	},
	{
		GetHash: "fff2525b8931402dd09222c50775608f75787bd2b87e56995a7bdd30f79702c4",
		txRaw:   "0100000001032e38e9c0a84c6046d687d10556dcacc41d275ec55fc00779ac88fdf357a187000000008c493046022100c352d3dd993a981beba4a63ad15c209275ca9470abfcd57da93b58e4eb5dce82022100840792bc1f456062819f15d33ee7055cf7b5ee1af1ebcc6028d9cdb1c3af7748014104f46db5e9d61a9dc27b8d64ad23e7383a4e6ca164593c2527c038c0857eb67ee8e825dca65046b82c9331586c82e0fd1f633f25f87c161bc6f8a630121df2b3d3ffffffff0200e32321000000001976a914c398efa9c392ba6013c5e04ee729755ef7f58b3288ac000fe208010000001976a914948c765a6914d43f2a7ac177da2c2f6b52de3d7c88ac00000000",
		tx: &Tx{
			version: 1,
			ins: []*txin.TxIn{
				txin.NewTxIn(outpoint.NewOutPoint(util.Hash{
					0x03, 0x2e, 0x38, 0xe9, 0xc0, 0xa8, 0x4c, 0x60,
					0x46, 0xd6, 0x87, 0xd1, 0x05, 0x56, 0xdc, 0xac,
					0xc4, 0x1d, 0x27, 0x5e, 0xc5, 0x5f, 0xc0, 0x07,
					0x79, 0xac, 0x88, 0xfd, 0xf3, 0x57, 0xa1, 0x87,
				}, 0), script.NewScriptRaw([]byte{
					0x49, // OP_DATA_73
					0x30, 0x46, 0x02, 0x21, 0x00, 0xc3, 0x52, 0xd3,
					0xdd, 0x99, 0x3a, 0x98, 0x1b, 0xeb, 0xa4, 0xa6,
					0x3a, 0xd1, 0x5c, 0x20, 0x92, 0x75, 0xca, 0x94,
					0x70, 0xab, 0xfc, 0xd5, 0x7d, 0xa9, 0x3b, 0x58,
					0xe4, 0xeb, 0x5d, 0xce, 0x82, 0x02, 0x21, 0x00,
					0x84, 0x07, 0x92, 0xbc, 0x1f, 0x45, 0x60, 0x62,
					0x81, 0x9f, 0x15, 0xd3, 0x3e, 0xe7, 0x05, 0x5c,
					0xf7, 0xb5, 0xee, 0x1a, 0xf1, 0xeb, 0xcc, 0x60,
					0x28, 0xd9, 0xcd, 0xb1, 0xc3, 0xaf, 0x77, 0x48,
					0x01, // 73-byte signature
					0x41, // OP_DATA_65
					0x04, 0xf4, 0x6d, 0xb5, 0xe9, 0xd6, 0x1a, 0x9d,
					0xc2, 0x7b, 0x8d, 0x64, 0xad, 0x23, 0xe7, 0x38,
					0x3a, 0x4e, 0x6c, 0xa1, 0x64, 0x59, 0x3c, 0x25,
					0x27, 0xc0, 0x38, 0xc0, 0x85, 0x7e, 0xb6, 0x7e,
					0xe8, 0xe8, 0x25, 0xdc, 0xa6, 0x50, 0x46, 0xb8,
					0x2c, 0x93, 0x31, 0x58, 0x6c, 0x82, 0xe0, 0xfd,
					0x1f, 0x63, 0x3f, 0x25, 0xf8, 0x7c, 0x16, 0x1b,
					0xc6, 0xf8, 0xa6, 0x30, 0x12, 0x1d, 0xf2, 0xb3,
					0xd3, // 65-byte pubkey
				}), 0xffffffff),
			},
			outs: []*txout.TxOut{
				txout.NewTxOut(0x2123e300, script.NewScriptRaw([]byte{
					0x76, // OP_DUP
					0xa9, // OP_HASH160
					0x14, // OP_DATA_20
					0xc3, 0x98, 0xef, 0xa9, 0xc3, 0x92, 0xba, 0x60,
					0x13, 0xc5, 0xe0, 0x4e, 0xe7, 0x29, 0x75, 0x5e,
					0xf7, 0xf5, 0x8b, 0x32,
					0x88, // OP_EQUALVERIFY
					0xac, // OP_CHECKSIG
				})),
				txout.NewTxOut(0x108e20f00, script.NewScriptRaw([]byte{
					0x76, // OP_DUP
					0xa9, // OP_HASH160
					0x14, // OP_DATA_20
					0x94, 0x8c, 0x76, 0x5a, 0x69, 0x14, 0xd4, 0x3f,
					0x2a, 0x7a, 0xc1, 0x77, 0xda, 0x2c, 0x2f, 0x6b,
					0x52, 0xde, 0x3d, 0x7c,
					0x88, // OP_EQUALVERIFY
					0xac, // OP_CHECKSIG
				})),
			},
		},
	},
}

func TestTxDeSerializeAndSerialize(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	for i, e := range tests {
		buf.Reset()
		b, err := hex.DecodeString(e.txRaw)
		if err != nil {
			t.Errorf("decode txRaw hex string :%v\n", err)
		}
		if _, err := buf.Write(b); err != nil {
			t.Errorf("write to buf :%v\n", err)
		}

		tx1 := Tx{}
		err = tx1.Unserialize(buf)
		if err != nil {
			t.Errorf("Deserialize error :%v\n", err)
		}

		buf.Reset()
		if err := e.tx.Serialize(buf); err != nil {
			t.Errorf("failed Serialize tx %d tx: %v\n", i, err)
		}
		h := util.DoubleSha256Hash(buf.Bytes())
		hash := h.ToString()
		if e.GetHash != hash {
			t.Errorf("failed compute GetHash for tx %d, expect=(%s), but got %s\n", i, e.GetHash, hash)
		}
	}
}

func TestTxSerializeAndTxUnserialize(t *testing.T) {
	rawTxString := "0100000002f6b52b137227b1992a6289e2c1b265953b559d782faba905c209ddf1c7a48fb8" +
		"000000006b48304502200a0c787cb9c132584e640b7686a8f6a78d9c4a41201a0c7a139d5383970b39c50" +
		"22100d6fdc88b87328cdd772ed4dd9f15fea84c85968fe84308bb4a207ba03889cd680121020b76df009c" +
		"b91ce792ae00461e15a9340652c30d1b816129fc61246b3441a9e2ffffffff9285678bbc575493ea20372" +
		"3507fa22e37d775e766eccadde8894fc561602f2c000000006a47304402201ba951afbdeda2cb70483aac" +
		"b144b1ddd9db6fdbe4b6ccaec27c005b4bc4048f0220549ac7d19ddb6c37852bfd23ca2ed4aef429f2432" +
		"e27b309bed1e9217ce68d03012102ee67fdeb2f4484a2342db30f851808942016ff8df57f7e7798a14710" +
		"fa761590ffffffff02b58d1300000000001976a914d8a6c89e4207a50d7f57c6a02fc09f38113ccb6b88a" +
		"c6c400a000000000017a914abaad350c1c83d2f33cc35e8c1c886cd1287bda98700000000"

	originBytes, err := hex.DecodeString(rawTxString)
	if err != nil {
		panic(err)
	}
	tx := Tx{}
	err = tx.Unserialize(bytes.NewReader(originBytes))
	testBytes := make([]byte, 64)
	hex.Encode(testBytes, tx.GetIns()[0].PreviousOutPoint.Hash[:])
	fmt.Printf("txIn preout hash: %x", testBytes)
	if err != nil {
		panic(err)
	}

	buf := bytes.NewBuffer(nil)
	err = tx.Serialize(buf)
	if err != nil {
		panic(err)
	}

	if !bytes.Equal(originBytes, buf.Bytes()) {
		t.Error("Serialize or Unserialize error")
	}
}
