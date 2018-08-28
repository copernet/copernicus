package db

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
)

func rand256() []byte {
	b := make([]byte, 256)
	rand.Read(b)
	return b
}

func TestDBWrapper(t *testing.T) {
	path, err := ioutil.TempDir("", "dbwtest")
	if err != nil {
		t.Fatalf("generate temp db path failed: %s\n", err)
	}
	defer os.RemoveAll(path)

	dbw, err := NewDBWrapper(&DBOption{
		FilePath:  path,
		CacheSize: 1 << 20,
	})
	if err != nil {
		t.Fatalf("NewDBWrapper failed: %s\n", err)
	}
	defer dbw.Close()

	key := []byte{'k'}
	in := rand256()
	if err := dbw.Write(key, in, false); err != nil {
		t.Fatalf("dbw.Write(): %s", err)
	}
	val, err := dbw.Read(key)
	if err != nil {
		t.Fatalf("dbw.Read(): %s", err)
	}
	if !bytes.Equal(in, val) {
		t.Fatalf("should read back original data")
	}

}

func TestDBWrapperBatch(t *testing.T) {
	path, err := ioutil.TempDir("", "dbwtest")
	if err != nil {
		t.Fatalf("generate temp db path failed: %s\n", err)
	}
	defer os.RemoveAll(path)

	dbw, err := NewDBWrapper(&DBOption{
		FilePath:  path,
		CacheSize: 1 << 20,
	})
	if err != nil {
		t.Fatalf("NewDBWrapper failed: %s\n", err)
	}
	defer dbw.Close()

	key := []byte{'i'}
	key2 := []byte{'j'}
	key3 := []byte{'k'}
	in := rand256()
	in2 := rand256()
	in3 := rand256()

	batch := NewBatchWrapper(dbw)
	batch.Write(key, in)
	batch.Write(key2, in2)
	batch.Write(key3, in3)

	batch.Erase(key3)
	dbw.WriteBatch(batch, false)

	res, err := dbw.Read(key)
	if err != nil {
		t.Fatalf("dbw.Read(): %s", err)
	}
	if !bytes.Equal(res, in) {
		t.Fatalf("should read back key 'i' value")
	}

	res, err = dbw.Read(key2)
	if err != nil {
		t.Fatalf("dbw.Read(): %s", err)
	}
	if !bytes.Equal(res, in2) {
		t.Fatalf("should read back key 'j' value")
	}

	if dbw.Exists(key3) {
		t.Fatalf("shouldn't read out key 'k' value")
	}

}

func TestDBWrapperIterator(t *testing.T) {
	path, err := ioutil.TempDir("", "dbwtest")
	if err != nil {
		t.Fatalf("generate temp db path failed: %s\n", err)
	}
	defer os.RemoveAll(path)

	dbw, err := NewDBWrapper(&DBOption{
		FilePath:  path,
		CacheSize: 1 << 20,
	})
	if err != nil {
		t.Fatalf("NewDBWrapper failed: %s\n", err)
	}
	defer dbw.Close()

	key := []byte{'j'}
	in := rand256()
	if err := dbw.Write(key, in, false); err != nil {
		t.Fatalf("dbw.Write(): %s", err)
	}

	key2 := []byte{'k'}
	in2 := rand256()
	if err := dbw.Write(key2, in2, false); err != nil {
		t.Fatalf("dbw.Write(): %s", err)
	}

	iter := dbw.Iterator()
	defer iter.Close()

	iter.Seek(key)
	if !bytes.Equal(iter.GetKey(), key) {
		t.Fatalf("iter.GetKey() should read back key 'j'")
	}
	if !bytes.Equal(iter.GetVal(), in) {
		t.Fatalf("iter.GetVal() should read back key 'j' value")
	}

	iter.Next()

	if !bytes.Equal(iter.GetKey(), key2) {
		t.Fatalf("iter.GetKey() should read back key 'k'")
	}
	if !bytes.Equal(iter.GetVal(), in2) {
		t.Fatalf("iter.GetVal() should read back key 'k' value")
	}

	iter.Next()
	if iter.Valid() {
		t.Fatalf("now iter should be invalid")
	}

}

func TestExistingDataNoObfuscate(t *testing.T) {
	path, err := ioutil.TempDir("", "dbwtest")
	if err != nil {
		t.Fatalf("generate temp db path failed: %s\n", err)
	}
	defer os.RemoveAll(path)

	dbw, err := NewDBWrapper(&DBOption{
		FilePath:  path,
		CacheSize: 1 << 10,
	})
	if err != nil {
		t.Fatalf("NewDBWrapper failed: %s\n", err)
	}

	key := []byte{'k'}
	in := rand256()
	if err := dbw.Write(key, in, false); err != nil {
		t.Fatalf("dbw.Write(): %s", err)
	}
	if res, err := dbw.Read(key); err != nil {
		t.Fatalf("dbw.Read(): %s", err)
	} else if err == nil && !bytes.Equal(res, in) {
		t.Fatalf("res should equal in")
	}

	dbw.Close()

	odbw, err := NewDBWrapper(&DBOption{
		FilePath:  path,
		CacheSize: 1 << 10,
	})
	if err != nil {
		t.Fatalf("NewDBWrapper failed: %s\n", err)
	}
	defer odbw.Close()

	if res, err := odbw.Read(key); err != nil {
		t.Fatalf("dbw.Read(): %s", err)
	} else if err == nil && !bytes.Equal(res, in) {
		t.Fatalf("res should equal in")
	}
	if odbw.IsEmpty() {
		t.Fatalf("There should be existing data")
	}

	in2 := rand256()
	if err := odbw.Write(key, in2, false); err != nil {
		t.Fatalf("dbw.Write(): %s", err)
	}
	if res, err := odbw.Read(key); err != nil {
		t.Fatalf("dbw.Read(): %s", err)
	} else if err == nil && !bytes.Equal(res, in2) {
		t.Fatalf("res should equal in2")
	}
}

func TestExistingDataReindex(t *testing.T) {
	path, err := ioutil.TempDir("", "dbwtest")
	if err != nil {
		t.Fatalf("generate temp db path failed: %s\n", err)
	}
	defer os.RemoveAll(path)

	dbw, err := NewDBWrapper(&DBOption{
		FilePath:  path,
		CacheSize: 1 << 10,
		//DontObfuscate: true,
	})
	if err != nil {
		t.Fatalf("NewDBWrapper failed: %s\n", err)
	}

	key := []byte{'k'}
	in := rand256()
	if err := dbw.Write(key, in, false); err != nil {
		t.Fatalf("dbw.Write(): %s", err)
	}
	if res, err := dbw.Read(key); err != nil {
		t.Fatalf("dbw.Read(): %s", err)
	} else if err == nil && !bytes.Equal(res, in) {
		t.Fatalf("res should equal in")
	}

	dbw.Close()

	odbw, err := NewDBWrapper(&DBOption{
		FilePath:  path,
		CacheSize: 1 << 10,
		Wipe: true,
	})
	if err != nil {
		t.Fatalf("NewDBWrapper failed: %s\n", err)
	}
	defer odbw.Close()

	if odbw.Exists(key) {
		t.Fatalf("odbw should not contain 'k'")
	}

	in2 := rand256()
	if err := odbw.Write(key, in2, false); err != nil {
		t.Fatalf("dbw.Write(): %s", err)
	}
	if res, err := odbw.Read(key); err != nil {
		t.Fatalf("dbw.Read(): %s", err)
	} else if err == nil && !bytes.Equal(res, in2) {
		t.Fatalf("res should equal in2")
	}
}

func TestIteratorOrdering(t *testing.T) {
	path, err := ioutil.TempDir("", "dbwtest")
	if err != nil {
		t.Fatalf("generate temp db path failed: %s\n", err)
	}
	defer os.RemoveAll(path)

	dbw, err := NewDBWrapper(&DBOption{
		FilePath:  path,
		CacheSize: 1 << 20,
	})
	if err != nil {
		t.Fatalf("NewDBWrapper failed: %s\n", err)
	}
	defer dbw.Close()

	for i := 0; i < 256; i++ {
		key := uint8(i)
		val := uint32(i * i)
		if i&1 == 0 {
			vs := make([]byte, 4)
			binary.LittleEndian.PutUint32(vs, val)
			if err := dbw.Write([]byte{key}, vs, false); err != nil {
				t.Fatalf("dbw.Write(): %s", err)
			}
		}
	}
	iter := dbw.Iterator()
	defer iter.Close()
	for i := 0; i < 256; i++ {
		key := uint8(i)
		val := uint32(i * i)
		if i&1 != 0 {
			vs := make([]byte, 4)
			binary.LittleEndian.PutUint32(vs, val)
			if err := dbw.Write([]byte{key}, vs, false); err != nil {
				t.Fatalf("dbw.Write(): %s", err)
			}
		}
	}

	for _, seekStart := range []byte{0x00, 0x80} {
		iter.Seek([]byte{seekStart})
		for x := uint32(seekStart); x < 0xff; x++ {
			k := uint32(0)
			v := uint32(0)
			if !iter.Valid() {
				t.Fatalf("iter should be valid")
			}
			if !iter.Valid() {
				break
			}
			ks := iter.GetKey()
			if len(ks) == 0 {
				t.Fatalf("iter.GetKey() should return non empty key")
			}
			k = uint32(ks[0])

			if x&1 != 0 {
				if k != x+1 {
					t.Fatal("k should equal x + 1")
				}
				continue
			}
			v = binary.LittleEndian.Uint32(iter.GetVal())

			if k != x {
				t.Fatalf("key should equal x")
			}
			if v != x*x {
				t.Fatalf("value should equal x*x")
			}
			iter.Next()
		}
		if iter.Valid() {
			t.Fatalf("iterator now should be invalid")
		}
	}
}

func TestIteratorStringOrdering(t *testing.T) {
	path, err := ioutil.TempDir("", "dbwtest")
	if err != nil {
		t.Fatalf("generate temp db path failed: %s\n", err)
	}
	defer os.RemoveAll(path)

	dbw, err := NewDBWrapper(&DBOption{
		FilePath:  path,
		CacheSize: 1 << 20,
	})
	if err != nil {
		t.Fatalf("NewDBWrapper failed: %s\n", err)
	}
	defer dbw.Close()

	for x := 0; x < 10; x++ {
		for y := 0; y < 10; y++ {
			key := bytes.NewBuffer(nil)
			key.WriteString(fmt.Sprintf("%d", x))
			for z := 0; z < y; z++ {
				key.Write(key.Bytes())
			}
			val := make([]byte, 4)
			binary.LittleEndian.PutUint32(val, uint32(x*x))
			if err := dbw.Write(key.Bytes(), val, false); err != nil {
				t.Fatalf("dbw.Write(): %s", err)
			}
		}
	}

	iter := dbw.Iterator()
	defer iter.Close()

	for _, seekStart := range []int{0, 5} {
		iter.Seek([]byte(fmt.Sprintf("%d", seekStart)))
		for x := seekStart; x < 10; x++ {
			for y := 0; y < 10; y++ {
				expKey := bytes.NewBuffer(nil)
				expKey.WriteString(fmt.Sprintf("%d", x))
				for z := 0; z < y; z++ {
					expKey.Write(expKey.Bytes())
				}
				if !iter.Valid() {
					t.Fatalf("iter should be valid")
				}
				if !iter.Valid() {
					break
				}
				ks := iter.GetKey()
				vs := iter.GetVal()
				if len(ks) == 0 || len(vs) == 0 {
					t.Fatal("ks or vs should not be empty")
				}
				t.Logf("expKey=%s, ks=%s, val=%d\n", expKey.Bytes(), ks, binary.LittleEndian.Uint32(vs))
				if !bytes.Equal(expKey.Bytes(), ks) {
					t.Fatal("expKey should equal ks")
				}
				if binary.LittleEndian.Uint32(vs) != uint32(x*x) {
					t.Fatal("value should equal x * x")
				}
				iter.Next()
			}
		}
		if iter.Valid() {
			t.Fatalf("iterator now should be invalid")
		}
	}
}
