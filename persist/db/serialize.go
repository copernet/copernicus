package db

//
//
//type ISerialize interface{
//	GetSerializeList()[]string
//	Unserialize(r io.Reader) error
//	Serialize(w io.Writer) error
//}
//type ISerializable interface{
//	Unserialize(r io.Reader) error
//	Serialize(w io.Writer) error
//}
//
//func SerializeOP(w io.Writer, obj ISerialize) error{
//	dumpList := obj.GetSerializeList()
//	rv := reflect.ValueOf(obj)
//	var err error
//	for _, key := range dumpList{
//
//		value := rv.FieldByName(key)
//		v := value.Interface()
//		switch v.(type){
//		case uint32:
//			err = util.WriteVarLenInt(w, value.Uint())
//		case uint64:
//			err = util.WriteVarLenInt(w, value.Uint())
//		case ISerialize:
//			vv, _ := v.(ISerialize)
//			SerializeOP(w,vv)
//		case ISerializable:
//			vv, _ := v.(ISerializable)
//			vv.Serialize(w)
//		default:
//			logs.Alert("SerializeOP:value.Interface().(type)======%#v, %#v",value,obj)
//			panic("SerializeOP.err====")
//		}
//
//
//		if err!= nil{
//			return err
//		}
//	}
//	return err
//}
//
//
//func UnserializeOP(r io.Reader, obj ISerialize) error{
//	rv := reflect.ValueOf(obj)
//	dumpList := obj.GetSerializeList()
//	var err error
//	for _, key := range dumpList{
//		field := rv.FieldByName(key)
//		switch field.Kind(){
//		case reflect.Uint32:
//			var v uint32
//			v, err = util.BinarySerializer.Uint32(r, binary.LittleEndian)
//			if err!= nil{
//				return err
//			}
//			field.Set(reflect.ValueOf(v))
//		case reflect.Uint64:
//			var v uint64
//			v, err = util.BinarySerializer.Uint64(r, binary.LittleEndian)
//			if err!= nil{
//				return err
//			}
//			field.Set(reflect.ValueOf(v))
//		default:
//			err = errors.New("can not handle field type")
//			panic("SerializeOP.err====")
//
//			return err
//		}
//
//	}
//	return err
//}
