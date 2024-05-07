package nbt

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"reflect"
	"unsafe"
)

const (
	TagEnd byte = iota
	TagByte
	TagShort
	TagInt
	TagLong
	TagFloat
	TagDouble
	TagByteArray
	TagString
	TagList
	TagCompound
	TagIntArray
	TagLongArray
)

var (
	ErrTypeMismatch = errors.New("Type mismatch")
)

type Decoder struct {
	r     *bufio.Reader
	limit int
}

func NewDecoder(r io.Reader) *Decoder {
	br, ok := r.(*bufio.Reader)
	if !ok {
		br = bufio.NewReader(r)
	}

	return &Decoder{r: br}
}

func NewDecoderWithDepthLimit(r io.Reader, limit int) *Decoder {
	decoder := NewDecoder(r)
	decoder.limit = limit
	return decoder
}

func getStructFields(ty reflect.Type) map[string]int {
	fields := make(map[string]int)
	for i := 0; i < ty.NumField(); i++ {
		field := ty.Field(i)
		name := field.Tag.Get("nbt")
		if name == "" {
			name = field.Name
		}
		fields[name] = i
	}
	return fields
}

func (dec *Decoder) readByte(v *reflect.Value, level int) error {
	value, err := dec.r.ReadByte()
	if err != nil {
		return err
	}
	if v != nil {
		if !v.CanInt() {
			return ErrTypeMismatch
		}
		v.SetInt(int64(int8(value)))
	}
	return nil
}

func (dec *Decoder) readShort(v *reflect.Value, level int) error {
	buf := make([]byte, 2)
	if _, err := io.ReadFull(dec.r, buf); err != nil {
		return err
	}
	if v != nil {
		if !v.CanInt() {
			return ErrTypeMismatch
		}
		value := int64(int16(binary.BigEndian.Uint16(buf)))
		v.SetInt(value)
	}
	return nil
}

func (dec *Decoder) readInt(v *reflect.Value, level int) error {
	buf := make([]byte, 4)
	if _, err := io.ReadFull(dec.r, buf); err != nil {
		return err
	}
	if v != nil {
		if !v.CanInt() {
			return ErrTypeMismatch
		}
		value := int64(int32(binary.BigEndian.Uint32(buf)))
		v.SetInt(value)
	}
	return nil
}

func (dec *Decoder) readLong(v *reflect.Value, level int) error {
	buf := make([]byte, 8)
	if _, err := io.ReadFull(dec.r, buf); err != nil {
		return err
	}
	if v != nil {
		if !v.CanInt() {
			return ErrTypeMismatch
		}
		value := int64(binary.BigEndian.Uint64(buf))
		v.SetInt(value)
	}
	return nil
}

func (dec *Decoder) readFloat(v *reflect.Value, level int) error {
	buf := make([]byte, 4)
	if _, err := io.ReadFull(dec.r, buf); err != nil {
		return err
	}
	if v != nil {
		if !v.CanFloat() {
			return ErrTypeMismatch
		}
		tmp := binary.BigEndian.Uint32(buf)
		binary.LittleEndian.PutUint32(buf, tmp)
		v.SetFloat(float64(*(*float32)(unsafe.Pointer(&buf[0]))))
	}
	return nil
}

func (dec *Decoder) readDouble(v *reflect.Value, level int) error {
	buf := make([]byte, 8)
	if _, err := io.ReadFull(dec.r, buf); err != nil {
		return err
	}
	if v != nil {
		if !v.CanFloat() {
			return ErrTypeMismatch
		}
		tmp := binary.BigEndian.Uint64(buf)
		binary.LittleEndian.PutUint64(buf, tmp)
		v.SetFloat(*(*float64)(unsafe.Pointer(&buf[0])))
	}
	return nil
}

func (dec *Decoder) readByteArray(v *reflect.Value, level int) error {
	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(dec.r, lenBuf); err != nil {
		return err
	}
	arrayLen := int(binary.BigEndian.Uint32(lenBuf))

	var result []byte
	for i := 0; i < arrayLen; i++ {
		value, err := dec.r.ReadByte()
		if err != nil {
			return err
		}
		result = append(result, value)
	}
	if v != nil {
		if v.Type() != reflect.SliceOf(reflect.TypeOf(byte(0))) {
			return ErrTypeMismatch
		}
		v.Set(reflect.ValueOf(result))
	}
	return nil
}

func (dec *Decoder) readString(v *reflect.Value, level int) error {
	lenBuf := make([]byte, 2)
	if _, err := io.ReadFull(dec.r, lenBuf); err != nil {
		return err
	}
	valueLen := binary.BigEndian.Uint16(lenBuf)

	value := make([]byte, int(valueLen))
	if _, err := io.ReadFull(dec.r, value); err != nil {
		return err
	}
	if v != nil {
		if v.Kind() != reflect.String {
			return ErrTypeMismatch
		}
		v.SetString(string(value))
	}
	return nil
}

func (dec *Decoder) readList(v *reflect.Value, level int) error {
	ty, err := dec.r.ReadByte()
	if err != nil {
		return err
	}

	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(dec.r, lenBuf); err != nil {
		return err
	}
	listLen := int(binary.BigEndian.Uint32(lenBuf))

	if listLen == 0 {
		// Minecraft can generate lists of TAG_End with length 0.
		// We don't care about elements' type if it's an empty list.
		return nil
	}

	typeReader, err := dec.getTypeReader(ty)
	if err != nil {
		return err
	}

	if v == nil {
		for i := 0; i < listLen; i++ {
			if err := typeReader(nil, level); err != nil {
				return err
			}
		}
	} else {
		if v.Kind() != reflect.Slice {
			return ErrTypeMismatch
		}
		elemType := v.Type().Elem()
		slice := reflect.MakeSlice(reflect.SliceOf(elemType), 0, 0)
		for i := 0; i < listLen; i++ {
			v := reflect.New(elemType).Elem()
			if err := typeReader(&v, level); err != nil {
				return err
			}
			slice = reflect.Append(slice, v)
		}
		v.Set(slice)
	}

	return nil
}

func (dec *Decoder) readCompound(v *reflect.Value, level int) error {
	if dec.limit > 0 && level >= dec.limit {
		return errors.New("Level limit exceeded")
	}

	if v != nil && v.Kind() != reflect.Struct {
		return errors.New("TagCompound can only unmarshaled into a struct")
	}

	fields := make(map[string]int)
	if v != nil {
		fields = getStructFields(v.Type())
	}

	for {
		ty, err := dec.r.ReadByte()
		if err != nil {
			return err
		}

		if ty == TagEnd {
			break
		}

		name, err := dec.readName()
		if err != nil {
			return err
		}

		typeReader, err := dec.getTypeReader(ty)
		if err != nil {
			return err
		}

		var value *reflect.Value
		if idx, ok := fields[name]; ok {
			tmp := v.Field(idx)
			value = &tmp
		}

		if err := typeReader(value, level+1); err != nil {
			return err
		}
	}

	return nil
}

func (dec *Decoder) readIntArray(v *reflect.Value, level int) error {
	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(dec.r, lenBuf); err != nil {
		return err
	}
	arrayLen := int(binary.BigEndian.Uint32(lenBuf))

	var result []int32
	buf := make([]byte, 4)
	for i := 0; i < arrayLen; i++ {
		if _, err := io.ReadFull(dec.r, buf); err != nil {
			return err
		}
		result = append(result, int32(binary.BigEndian.Uint32(buf)))
	}
	if v != nil {
		if v.Type() != reflect.SliceOf(reflect.TypeOf(int32(0))) {
			return ErrTypeMismatch
		}
		v.Set(reflect.ValueOf(result))
	}
	return nil
}

func (dec *Decoder) readLongArray(v *reflect.Value, level int) error {
	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(dec.r, lenBuf); err != nil {
		return err
	}
	arrayLen := int(binary.BigEndian.Uint32(lenBuf))

	var result []int64
	buf := make([]byte, 8)
	for i := 0; i < arrayLen; i++ {
		if _, err := io.ReadFull(dec.r, buf); err != nil {
			return err
		}
		result = append(result, int64(binary.BigEndian.Uint64(buf)))
	}
	if v != nil {
		if v.Type() != reflect.SliceOf(reflect.TypeOf(int64(0))) {
			return ErrTypeMismatch
		}
		v.Set(reflect.ValueOf(result))
	}
	return nil
}

func (dec *Decoder) readName() (string, error) {
	nameLenBuf := make([]byte, 2)
	if _, err := io.ReadFull(dec.r, nameLenBuf); err != nil {
		return "", err
	}
	nameLen := binary.BigEndian.Uint16(nameLenBuf)

	name := make([]byte, int(nameLen))
	if _, err := io.ReadFull(dec.r, name); err != nil {
		return "", err
	}
	return string(name), nil
}

func (dec *Decoder) getTypeReader(ty byte) (func(*reflect.Value, int) error, error) {
	switch ty {
	case TagEnd:
		return nil, errors.New("Invalid TAG_End")
	case TagByte:
		return dec.readByte, nil
	case TagShort:
		return dec.readShort, nil
	case TagInt:
		return dec.readInt, nil
	case TagLong:
		return dec.readLong, nil
	case TagFloat:
		return dec.readFloat, nil
	case TagDouble:
		return dec.readDouble, nil
	case TagByteArray:
		return dec.readByteArray, nil
	case TagString:
		return dec.readString, nil
	case TagList:
		return dec.readList, nil
	case TagCompound:
		return dec.readCompound, nil
	case TagIntArray:
		return dec.readIntArray, nil
	case TagLongArray:
		return dec.readLongArray, nil
	default:
		return nil, fmt.Errorf("Invalid tag type: %d", ty)
	}
}

func (dec *Decoder) decode(v reflect.Value, level int) error {
	nbtTy, err := dec.r.ReadByte()
	if err != nil {
		return err
	}

	// Dispose tag name of top tag
	dec.readName()

	typeReader, err := dec.getTypeReader(nbtTy)
	if err != nil {
		return err
	}

	return typeReader(&v, level)
}

func (dec *Decoder) Decode(v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return errors.New("v must be a pointer and not be nil")
	}

	return dec.decode(rv.Elem(), 0)
}
