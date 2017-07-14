package protocol

import (
	"encoding/binary"
	"github.com/btcboost/copernicus/btcutil"
	"github.com/btcboost/copernicus/utils"
	"io"
	"time"
)

const CommandSize = 12

func ReadElement(reader io.Reader, element interface{}) error {
	switch e := element.(type) {
	case *uint16:
		rv, err := utils.BinarySerializer.Uint16(reader, binary.LittleEndian)
		if err != nil {
			return err
		}
		*e = rv
		return nil
	case int32:
		rv, err := utils.BinarySerializer.Uint32(reader, binary.LittleEndian)
		if err != nil {
			return err
		}
		e = int32(rv)
		return nil
	case *uint32:
		rv, err := utils.BinarySerializer.Uint32(reader, binary.LittleEndian)
		if err != nil {
			return err
		}
		*e = rv
		return nil
	case *int64:
		rv, err := utils.BinarySerializer.Uint64(reader, binary.LittleEndian)
		if err != nil {
			return err
		}
		*e = int64(rv)
		return nil
	case *uint64:
		rv, err := utils.BinarySerializer.Uint64(reader, binary.LittleEndian)
		if err != nil {
			return err
		}
		*e = rv
		return nil
	case *bool:
		rv, err := utils.BinarySerializer.Uint8(reader)
		if err != nil {
			return err
		}
		if rv == 0x00 {
			*e = false
		} else {
			*e = true
		}
		return nil
	case *Uint32Time:
		rv, err := utils.BinarySerializer.Uint64(reader, binary.LittleEndian)
		if err != nil {
			return err
		}
		*e = Uint32Time(time.Unix(int64(rv), 0))
		return nil
	case *Int64Time:
		rv, err := utils.BinarySerializer.Uint64(reader, binary.LittleEndian)
		if err != nil {
			return err
		}
		*e = Int64Time(time.Unix(int64(rv), 0))
		return nil
		// Message header checksum.
	case *[4]byte:
		_, err := io.ReadFull(reader, e[:])
		if err != nil {
			return err
		}
		return nil
	case *[CommandSize]uint8:
		_, err := io.ReadFull(reader, e[:])
		if err != nil {
			return err
		}
		return nil
	case *utils.Hash:
		_, err := io.ReadFull(reader, e[:])
		if err != nil {
			return err
		}
		return nil
	case *ServiceFlag:
		rv, err := utils.BinarySerializer.Uint64(reader, binary.LittleEndian)
		if err != nil {
			return err
		}
		*e = ServiceFlag(rv)
		return nil
	case *InventoryType:
		rv, err := utils.BinarySerializer.Uint32(reader, binary.LittleEndian)
		if err != nil {
			return err
		}
		*e = InventoryType(rv)
		return nil
	case *btcutil.BitcoinNet:
		rv, err := utils.BinarySerializer.Uint32(reader, binary.LittleEndian)
		if err != nil {
			return err
		}
		*e = btcutil.BitcoinNet(rv)
		return nil

	case *BloomUpdateType:
		rv, err := utils.BinarySerializer.Uint8(reader)
		if err != nil {
			return err
		}
		*e = BloomUpdateType(rv)
		return nil
	case *RejectCode:
		rv, err := utils.BinarySerializer.Uint8(reader)
		if err != nil {
			return err
		}
		*e = RejectCode(rv)
		return nil

	}
	return binary.Read(reader, binary.LittleEndian, element)

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
		err := utils.BinarySerializer.PutUint32(w, binary.LittleEndian, uint32(e))
		if err != nil {
			return err
		}
		return nil
	case uint32:
		err := utils.BinarySerializer.PutUint32(w, binary.LittleEndian, e)
		if err != nil {
			return err
		}
		return nil

	case int64:
		err := utils.BinarySerializer.PutUint64(w, binary.LittleEndian, uint64(e))
		if err != nil {
			return err
		}
		return nil
	case uint64:
		err := utils.BinarySerializer.PutUint64(w, binary.LittleEndian, e)
		if err != nil {
			return err
		}
		return nil
	case bool:
		var err error
		if e {
			err = utils.BinarySerializer.PutUint8(w, 0x01)

		} else {
			err = utils.BinarySerializer.PutUint8(w, 0x00)
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
	case [CommandSize]uint8:
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
	case *utils.Hash:
		_, err := w.Write(e[:])
		if err != nil {
			return err
		}
		return nil
	case ServiceFlag:
		err := utils.BinarySerializer.PutUint64(w, binary.LittleEndian, uint64(e))
		if err != nil {
			return err
		}
		return nil
	case InventoryType:
		err := utils.BinarySerializer.PutUint32(w, binary.LittleEndian, uint32(e))
		if err != nil {
			return err
		}
		return nil
	case btcutil.BitcoinNet:
		err := utils.BinarySerializer.PutUint32(w, binary.LittleEndian, uint32(e))
		if err != nil {
			return err
		}
		return nil
	case BloomUpdateType:
		err := utils.BinarySerializer.PutUint8(w, uint8(e))
		if err != nil {
			return err
		}
		return nil
	case RejectCode:
		err := utils.BinarySerializer.PutUint8(w, uint8(e))
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
