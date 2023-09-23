package config

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
)

var (
	ErrUnsupportedType = errors.New("Unsupported field type")
)

var (
	isInDocker = false
)

const (
	ignoreFieldName = "-"
)

func init() {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		isInDocker = true
	}
}

func xGetenv(name string) string {
	val := os.Getenv(name)
	if isInDocker {
		if len(val) >= 2 && val[0] == '\'' && val[len(val)-1] == '\'' {
			return val[1 : len(val)-1]
		}
	}
	return val
}

func loadField(name string, field reflect.Value) error {
	switch field.Type().Kind() {
	case reflect.String:
		field.SetString(xGetenv(name))
		break

	case reflect.Int:
		result, err := strconv.ParseInt(xGetenv(name), 0, 64)
		if err != nil {
			return fmt.Errorf("Failed to parse %s: %w", name, err)
		}
		field.SetInt(result)
		break

	case reflect.Uint:
		result, err := strconv.ParseUint(xGetenv(name), 0, 64)
		if err != nil {
			return fmt.Errorf("Failed to parse %s: %w", name, err)
		}
		field.SetUint(result)
		break

	case reflect.Float32:
		fallthrough
	case reflect.Float64:
		result, err := strconv.ParseFloat(xGetenv(name), 64)
		if err != nil {
			return fmt.Errorf("Failed to parse %s: %w", name, err)
		}
		field.SetFloat(result)
		break

	case reflect.Bool:
		val := xGetenv(name)
		field.SetBool(strings.EqualFold(val, "true") || strings.EqualFold(val, "yes") || strings.EqualFold(val, "on"))
		break

	case reflect.Slice:
		sliceInterface := field.Interface()
		switch field.Type().Elem().Kind() {
		case reflect.String:
			slice := sliceInterface.([]string)
			for _, v := range strings.Split(xGetenv(name), ",") {
				slice = append(slice, v)
			}
			field.Set(reflect.ValueOf(slice))
			break

		case reflect.Int:
			slice := sliceInterface.([]int)
			for i, v := range strings.Split(xGetenv(name), ",") {
				val, err := strconv.ParseInt(v, 0, 64)
				if err != nil {
					return fmt.Errorf("Failed to parse %dth of %s: %w", i, name, err)
				}
				slice = append(slice, int(val))
			}
			field.Set(reflect.ValueOf(slice))
			break

		case reflect.Uint:
			slice := sliceInterface.([]uint)
			for i, v := range strings.Split(xGetenv(name), ",") {
				val, err := strconv.ParseUint(v, 0, 64)
				if err != nil {
					return fmt.Errorf("Failed to parse %dth of %s: %w", i, name, err)
				}
				slice = append(slice, uint(val))
			}
			field.Set(reflect.ValueOf(slice))
			break

		case reflect.Float32:
			slice := sliceInterface.([]float32)
			for i, v := range strings.Split(xGetenv(name), ",") {
				val, err := strconv.ParseFloat(v, 32)
				if err != nil {
					return fmt.Errorf("Failed to parse %dth of %s: %w", i, name, err)
				}
				slice = append(slice, float32(val))
			}
			field.Set(reflect.ValueOf(slice))
			break

		case reflect.Float64:
			slice := sliceInterface.([]float64)
			for i, v := range strings.Split(xGetenv(name), ",") {
				val, err := strconv.ParseFloat(v, 64)
				if err != nil {
					return fmt.Errorf("Failed to parse %dth of %s: %w", i, name, err)
				}
				slice = append(slice, float64(val))
			}
			field.Set(reflect.ValueOf(slice))
			break
		}
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
		fieldName, ok := fieldType.Tag.Lookup("env")
		if !ok {
			fieldName = strings.ToLower(fieldType.Name)
		} else if ok && fieldName == ignoreFieldName {
			continue
		}
		name := prefix + "_" + fieldName

		if field.Type().Kind() == reflect.Struct {
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

func loadToStruct(prefix string, v interface{}) error {
	elemType := reflect.TypeOf(v).Elem()
	elem := reflect.ValueOf(v).Elem()
	for i := 0; i < elem.NumField(); i++ {
		if !elemType.Field(i).IsExported() {
			continue
		}

		fieldType := elemType.Field(i)
		field := elem.Field(i)
		name, ok := fieldType.Tag.Lookup("env")
		if !ok {
			name = strings.ToLower(fieldType.Name)
		} else if ok && name == ignoreFieldName {
			continue
		}

		if prefix != "" {
			name = prefix + "_" + name
		}

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
