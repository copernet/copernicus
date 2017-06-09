package utils

import (
	"copernicus/protocol"
	"encoding/binary"
	"time"
	"io"
	"copernicus/crypto"
)

const COMMANG_SIZE = 12

func ReadElement(r io.Reader, element interface{}) (error) {

	switch e := element.(type) {
	case *uint16:
		rv, err := binarySerializer.Uint16(r, binary.LittleEndian)
		if err != nil {
			return err
		}
		*e = uint16(rv)
		return nil
	case *int32:
		rv, err := binarySerializer.Uint32(r, binary.LittleEndian)
		if err != nil {
			return err
		}
		*e = int32(rv)
		return nil
	case *uint32:
		rv, err := binarySerializer.Uint32(r, binary.LittleEndian)
		if err != nil {
			return err
		}
		*e = rv
		return nil
	case *int64:
		rv, err := binarySerializer.Uint64(r, binary.LittleEndian)
		if err != nil {
			return err
		}
		*e = int64(rv)
		return nil
	case *uint64:
		rv, err := binarySerializer.Uint64(r, binary.LittleEndian)
		if err != nil {
			return err
		}
		*e = rv
		return nil
	case *bool:
		rv, err := binarySerializer.Uint8(r)
		if err != nil {
			return err
		}
		if rv == 0x00 {
			*e = false
		} else {
			*e = true
		}
		return nil
	case *protocol.Uint32Time:
		rv, err := binarySerializer.Uint64(r, binary.LittleEndian)
		if err != nil {
			return err
		}
		*e = protocol.Uint32Time(time.Unix(int64(rv), 0))
		return nil
	case *protocol.Int64Time:
		rv, err := binarySerializer.Uint64(r, binary.LittleEndian)
		if err != nil {
			return err
		}
		*e = protocol.Int64Time(time.Unix(int64(rv), 0))
		return nil
		// Message header checksum.
	case *[4]byte:
		_, err := io.ReadFull(r, e[:])
		if err != nil {
			return err
		}
		return nil
	case *[COMMANG_SIZE]uint8:
		_, err := io.ReadFull(r, e[:])
		if err != nil {
			return err
		}
		return nil
	case *crypto.Hash:
		_, err := io.ReadFull(r, e[:])
		if err != nil {
			return err
		}
		return nil
	case *protocol.ServiceFlag:
		rv, err := binarySerializer.Uint64(r, binary.LittleEndian)
		if err != nil {
			return err
		}
		*e = protocol.ServiceFlag(rv)
		return nil
	case *protocol.InvType:
		rv, err := binarySerializer.Uint32(r, binary.LittleEndian)
		if err != nil {
			return err
		}
		*e = protocol.InvType(rv)
		return nil
	case *protocol.BitcoinNet:
		rv, err := binarySerializer.Uint32(r, binary.LittleEndian)
		if err != nil {
			return err
		}
		*e = protocol.BitcoinNet(rv)
		return nil

	case *protocol.BloomUpdateType:
		rv, err := binarySerializer.Uint8(r)
		if err != nil {
			return err
		}
		*e = protocol.BloomUpdateType(rv)
		return nil
	case *protocol.RejectCode:
		rv, err := binarySerializer.Uint8(r)
		if err != nil {
			return err
		}
		*e = protocol.RejectCode(rv)
		return nil

	}
	return binary.Read(r, binary.LittleEndian, element)

}

func ReadElements(r io.Reader, elements ...interface{}) error {
	for _, element := range elements {
		err := ReadElement(r, element)
		if err != nil {
			return err
		}
	}
	return nil
}
func WriteElement(w io.Writer, element interface{}) error {
	switch e := element.(type) {

	case int32:
		err := binarySerializer.PutUint32(w, binary.LittleEndian, uint32(e))
		if err != nil {
			return err
		}
		return nil
	case uint32:
		err := binarySerializer.PutUint32(w, binary.LittleEndian, e)
		if err != nil {
			return err
		}
		return nil

	case int64:
		err := binarySerializer.PutUint64(w, binary.LittleEndian, uint64(e))
		if err != nil {
			return err
		}
		return nil
	case uint64:
		err := binarySerializer.PutUint64(w, binary.LittleEndian, e)
		if err != nil {
			return err
		}
		return nil
	case bool:
		var err error
		if e {
			err = binarySerializer.PutUint8(w, 0x01)

		} else {
			err = binarySerializer.PutUint8(w, 0x00)
		}
		if err != nil {
			return err
		}
		return nil
	case [4]byte:
		_, err := w.Write(e[:])
		if err != nil {
			return err
		}
		return nil
	case [COMMANG_SIZE]uint8:
		_, err := w.Write(e[:])
		if err != nil {
			return err
		}
		return nil
		//ip address
	case [16]byte:
		_, err := w.Write(e[:])
		if err != nil {
			return err
		}
		return nil
	case *crypto.Hash:
		_, err := w.Write(e[:])
		if err != nil {
			return err
		}
		return nil
	case protocol.ServiceFlag:
		err := binarySerializer.PutUint64(w, binary.LittleEndian, uint64(e))
		if err != nil {
			return err
		}
		return nil
	case protocol.InvType:
		err := binarySerializer.PutUint32(w, binary.LittleEndian, uint32(e))
		if err != nil {
			return err
		}
		return nil
	case protocol.BitcoinNet:
		err := binarySerializer.PutUint32(w, binary.LittleEndian, uint32(e))
		if err != nil {
			return err
		}
		return nil
	case protocol.BloomUpdateType:
		err := binarySerializer.PutUint8(w, uint8(e))
		if err != nil {
			return err
		}
		return nil
	case protocol.RejectCode:
		err := binarySerializer.PutUint8(w, uint8(e))
		if err != nil {
			return err
		}
		return nil

	}
	return binary.Write(w, binary.LittleEndian, element)

}
func WriteElements(w io.Writer, elements ...interface{}) error {
	for _, element := range elements {
		err := WriteElement(w, element)
		if err != nil {
			return err
		}
	}
	return nil
}
