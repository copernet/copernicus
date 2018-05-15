package db

import (
	"reflect"
	"errors"
	"github.com/btcboost/copernicus/util"
	"encoding/binary"
	"io"
)

type Serializer struct {
}

func (s *Serializer) Serialize(w io.Writer) error {
	return SerializeOP(w, s)
}

func (s *Serializer) Unserialize(r io.Reader) error {
	err := UnserializeOP(r, s)
	return err
}

func (s *Serializer) GetSerializeList()[]string{
	panic("no impl of GetSerializeList ")
}
type ISerialize interface{
	GetSerializeList()[]string
	Unserialize(r io.Reader) error
	Serialize(w io.Writer) error
}


func SerializeOP(w io.Writer, obj ISerialize) error{
	dumpList := obj.GetSerializeList()
	rv := reflect.ValueOf(obj)
	var err error
	for _, key := range dumpList{
		value := rv.FieldByName(key)
		t := value.Type()
		switch value.Kind(){
		case reflect.Uint32:
			err = util.WriteVarLenInt(w, uint64(value.Uint()))
		case reflect.Uint64:
			err = util.WriteVarLenInt(w, uint64(value.Uint()))
		default:
			v, _ := t.(ISerialize)
			err = v.Serialize(w)
		}


		if err!= nil{
			return err
		}
	}
	return err
}


func UnserializeOP(r io.Reader, obj ISerialize) error{
	rv := reflect.ValueOf(obj)
	dumpList := obj.GetSerializeList()
	var err error
	for _, key := range dumpList{
		field := rv.FieldByName(key)
		switch field.Kind(){
		case reflect.Uint32:
			var v uint32
			v, err = util.BinarySerializer.Uint32(r, binary.LittleEndian)
			if err!= nil{
				return err
			}
			field.Set(reflect.ValueOf(v))
		case reflect.Uint64:
			var v uint64
			v, err = util.BinarySerializer.Uint64(r, binary.LittleEndian)
			if err!= nil{
				return err
			}
			field.Set(reflect.ValueOf(v))
		default:
			err = errors.New("can not handle field type")
			return err
		}

	}
	return err
}
