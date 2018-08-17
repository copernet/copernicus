package block

import (

	"testing"
	"os"
)

var testblockfileinfo = NewBlockFileInfo()
const (
	timefst    =uint64(1399703554)
	timelst	   =uint64(1513622125)
)



func TestNewBlockFileInfo(t *testing.T) {
	if testblockfileinfo.Blocks!=0 || testblockfileinfo.timeFirst!=0 || testblockfileinfo.timeLast != 0 ||
		testblockfileinfo.HeightFirst != 0 || testblockfileinfo.HeightLast != 0 || testblockfileinfo.Size != 0 ||
		testblockfileinfo.UndoSize != 0 || testblockfileinfo.index != 0 {
			t.Error("new BlockFileInfo not null")
	}
}


func TestSetNull(t *testing.T) {

	testblockfileinfo.SetNull()
	if testblockfileinfo.Blocks!=0 || testblockfileinfo.timeFirst!=0 || testblockfileinfo.timeLast != 0 ||
		testblockfileinfo.HeightFirst != 0 || testblockfileinfo.HeightLast != 0 || testblockfileinfo.Size != 0 ||
		testblockfileinfo.UndoSize != 0 {
		t.Error("function 'Setnull' out of order")
	}

	testblockfileinfo.Blocks = 20
	testblockfileinfo.index = 1
	testblockfileinfo.HeightFirst = 300000
	testblockfileinfo.HeightLast = 500000
	testblockfileinfo.timeFirst = timefst
	testblockfileinfo.timeLast = timelst
	testblockfileinfo.Size = 256
	testblockfileinfo.UndoSize = 60

	testblockfileinfo.SetNull()

	if testblockfileinfo.Blocks!=0 || testblockfileinfo.timeFirst!=0 || testblockfileinfo.timeLast != 0 ||
		testblockfileinfo.HeightFirst != 0 || testblockfileinfo.HeightLast != 0 || testblockfileinfo.Size != 0 ||
		testblockfileinfo.UndoSize != 0 {
		t.Error("function 'Setnull' out of order")
	}


}


func TestSetIndex(t *testing.T) {
	testblockfileinfo.SetIndex(4567743)
	i := testblockfileinfo.GetIndex()
	if i != 4567743 {
		t.Error("there exists sth. wrong in setting or getting")
	}
}

func TestGetSerializeList(t *testing.T) {
	var list = []string{"Blocks", "Size", "UndoSize", "HeightFirst", "HeightLast", "timeFirst", "timeLast", "index"}
	testlist := testblockfileinfo.GetSerializeList()

	for i:=0;i<8;i++ {
		if testlist[i] != list[i] {
			t.Error("wrong list")
			break
		}
	}

}


func TestSerialize(t *testing.T) {

	testblockfileinfo.Blocks = 20
	testblockfileinfo.index = 1
	testblockfileinfo.HeightFirst = 300000
	testblockfileinfo.HeightLast = 500000
	testblockfileinfo.timeFirst = timefst
	testblockfileinfo.timeLast = timelst
	testblockfileinfo.Size = 256
	testblockfileinfo.UndoSize = 60

	file, err := os.OpenFile("tmp.txt", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		t.Error(err)
	}

	err = testblockfileinfo.Serialize(file)
	if err != nil {
		t.Error(err)
	}

	_, err = file.Seek(0, 0)
	if err != nil {
		t.Error(err)
	}

	blockfileinforead := &BlockFileInfo{}
	blockfileinforead.SetNull()

	err = blockfileinforead.Unserialize(file)
	if err != nil {
		t.Error(err)
	}

	if blockfileinforead.Blocks != 20 {
		t.Error("the 'blocks' should be equal to the one of testblockfileinfo's but not")
	}

	if blockfileinforead.index != 1 {
		t.Error("the 'index' should be equal to the one of testblockfileinfo's but not")
	}

	if blockfileinforead.HeightFirst != 300000 {
		t.Error("the 'blocks' should be equal to the one of testblockfileinfo's but not")
	}

	if blockfileinforead.HeightLast != 500000 {
		t.Error("the 'blocks' should be equal to the one of testblockfileinfo's but not")
	}

	if blockfileinforead.timeFirst != timefst {
		t.Error("the 'blocks' should be equal to the one of testblockfileinfo's but not")
	}

	if blockfileinforead.timeLast != timelst {
		t.Error("the 'blocks' should be equal to the one of testblockfileinfo's but not")
	}

	if blockfileinforead.Size != 256 {
		t.Error("the 'blocks' should be equal to the one of testblockfileinfo's but not")
	}

	if blockfileinforead.UndoSize != 60 {
		t.Error("the 'blocks' should be equal to the one of testblockfileinfo's but not")
	}

	err = os.Remove("tmp.txt")
	if err != nil {
		t.Error(err)
	}
}


func TestAddBlock(t *testing.T) {

	testblockfileinfo.Blocks = 20
	testblockfileinfo.index = 1
	testblockfileinfo.HeightFirst = 300000
	testblockfileinfo.HeightLast = 500000
	testblockfileinfo.timeFirst = timefst
	testblockfileinfo.timeLast = timelst
	testblockfileinfo.Size = 256
	testblockfileinfo.UndoSize = 60

	heightofadd := int32(400000)
	timeofadd := uint64(1400000000)

	testblockfileinfo.AddBlock(heightofadd,timeofadd)

	if testblockfileinfo.Blocks != 21 {
		t.Error("num of blocks didn't += 1")
	}

	if testblockfileinfo.HeightFirst != 300000 || testblockfileinfo.HeightLast != 500000 {
		t.Error("operation of height made mistakes")
	}

	if testblockfileinfo.timeFirst != timefst || testblockfileinfo.timeLast != timelst {
		t.Error("operation of time made mistakes")
	}

	heightofadd = int32(200000)
	testblockfileinfo.AddBlock(heightofadd,timeofadd)
	if testblockfileinfo.Blocks != 22 {
		t.Error("num of blocks didn't += 1")
	}

	if testblockfileinfo.HeightFirst != 200000 || testblockfileinfo.HeightLast != 500000 {
		t.Error("operation of height made mistakes")
	}

	timeofadd = uint64(1600000000)
	testblockfileinfo.AddBlock(heightofadd,timeofadd)
	if testblockfileinfo.Blocks != 23 {
		t.Error("num of blocks didn't += 1")
	}
	if testblockfileinfo.timeFirst != timefst || testblockfileinfo.timeLast != 1600000000 {
		t.Error("operation of time made mistakes")
	}

}

func TestString(t *testing.T) {

	testblockfileinfo.SetNull()
	if testblockfileinfo.String() != "BlockFileInfo(blocks=0, size=0, heights=0...0," +
		" time=1970-01-01T08:00:00+08:00...1970-01-01T08:00:00+08:00)" {
			t.Error("initial state of testblockfileinfo is wrongly stringed")

	}

	testblockfileinfo.Blocks = 20
	testblockfileinfo.index = 1
	testblockfileinfo.HeightFirst = 300000
	testblockfileinfo.HeightLast = 500000
	testblockfileinfo.timeFirst = timefst
	testblockfileinfo.timeLast = timelst
	testblockfileinfo.Size = 256
	testblockfileinfo.UndoSize = 60

	if testblockfileinfo.String() != "BlockFileInfo(blocks=20, size=256, heights=300000...500000," +
		" time=2014-05-10T14:32:34+08:00...2017-12-19T02:35:25+08:00)" {
		t.Error("state of testblockfileinfo is wrongly stringed")
	}
}


