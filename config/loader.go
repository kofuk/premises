package config

import (
	"errors"
	"os"
	"reflect"
	"strconv"
	"strings"
)

var (
	ErrUnsupportedType = errors.New("Unsupported field type")
)

func loadField(name string, field reflect.Value) error {
	switch field.Type().Kind() {
	case reflect.String:
		field.SetString(os.Getenv(name))
		break

	case reflect.Int:
		fallthrough
	case reflect.Int8:
		fallthrough
	case reflect.Int16:
		fallthrough
	case reflect.Int32:
		fallthrough
	case reflect.Int64:

		result, err := strconv.ParseInt(os.Getenv(name), 0, 64)
		if err != nil {
			return err
		}
		field.SetInt(result)
		break

	case reflect.Uint:
		fallthrough
	case reflect.Uint8:
		fallthrough
	case reflect.Uint16:
		fallthrough
	case reflect.Uint32:
		fallthrough
	case reflect.Uint64:
		result, err := strconv.ParseUint(os.Getenv(name), 0, 64)
		if err != nil {
			return err
		}
		field.SetUint(result)
		break

	case reflect.Float32:
		fallthrough
	case reflect.Float64:
		result, err := strconv.ParseFloat(os.Getenv(name), 64)
		if err != nil {
			return err
		}
		field.SetFloat(result)
		break

	case reflect.Bool:
		result, err := strconv.ParseBool(os.Getenv(name))
		if err != nil {
			return err
		}
		field.SetBool(result)
		break

	default:
		return ErrUnsupportedType
	}

	return nil
}

func loadInnerField(prefix string, val reflect.Value, ty reflect.Type) error {
	for i := 0; i < val.NumField(); i++ {
		if !ty.Field(i).IsExported() {
			continue
		}

		field := val.Field(i)
		fieldType := ty.Field(i)
		name := prefix + "." + strings.ToLower(fieldType.Name)

		if field.Type().Kind() == reflect.Struct {
			if err := loadInnerField(name, reflect.ValueOf(field), ty.Field(i).Type); err != nil {
				return err
			}
			continue
		}

		if err := loadField(name, field); err != nil {
			return err
		}
	}

	return nil
}

func loadToStruct(v interface{}) error {
	elemType := reflect.TypeOf(v).Elem()
	elem := reflect.ValueOf(v).Elem()
	for i := 0; i < elem.NumField(); i++ {
		if !elemType.Field(i).IsExported() {
			continue
		}

		fieldType := elemType.Field(i)
		field := elem.Field(i)
		name := strings.ToLower(elemType.Field(i).Name)

		if fieldType.Type.Kind() == reflect.Struct {
			if err := loadInnerField(name, field, fieldType.Type); err != nil {
				return err
			}
			continue
		}

		if err := loadField(name, field); err != nil {
			return err
		}
	}

	return nil
}
