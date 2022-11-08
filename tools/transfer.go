package tools

import (
	"fmt"
	"reflect"
	"strconv"
)

func ToInt(val any) (int, error) {
	var t2 int
	var err error

	switch val.(type) {
	case uint:
		t2 = int(val.(uint))
	case int8:
		t2 = int(val.(int8))
	case uint8:
		t2 = int(val.(uint8))
	case int16:
		t2 = int(val.(int16))
	case uint16:
		t2 = int(val.(uint16))
	case int32:
		t2 = int(val.(int32))
	case uint32:
		t2 = int(val.(uint32))
	case int64:
		t2 = int(val.(int64))
	case uint64:
		t2 = int(val.(uint64))
	case float32:
		t2 = int(val.(float32))
	case float64:
		t2 = int(val.(float64))
	case string:
		t2, err = strconv.Atoi(val.(string))
	case int:
		t2 = val.(int)
	case bool:
		if val.(bool) == true {
			return 1, nil
		} else {
			return 0, nil
		}
	default:
		err = fmt.Errorf("%v transfer type to integer error", val)
	}
	return t2, err
}

func ToFloat(val any) (float64, error) {
	var t2 float64
	var err error
	switch val.(type) {
	case uint:
		t2 = float64(val.(uint))
	case int8:
		t2 = float64(val.(int8))
	case uint8:
		t2 = float64(val.(uint8))
	case int16:
		t2 = float64(val.(int16))
	case uint16:
		t2 = float64(val.(uint16))
	case int32:
		t2 = float64(val.(int32))
	case uint32:
		t2 = float64(val.(uint32))
	case int64:
		t2 = float64(val.(int64))
	case uint64:
		t2 = float64(val.(uint64))
	case float32:
		t2 = float64(val.(float32))
	case string:
		t2, err = strconv.ParseFloat(val.(string), 64)
	case float64:
		t2 = val.(float64)
	case bool:
		if val.(bool) == true {
			return 1.0, nil
		} else {
			return 0., nil
		}
	default:
		err = fmt.Errorf("%v transfer type to float error", val)
	}

	return t2, err
}

func ToString(val any) (string, error) {
	var t string
	var err error

	switch val.(type) {
	case uint8, uint16, uint32, uint, uint64, int8, int16, int32, int, int64, float64, float32, bool:
		t = fmt.Sprint(val)
	case string:
		t = val.(string)
	default:
		tp := reflect.TypeOf(val)
		if tp.Kind() == reflect.Slice || tp.Kind() == reflect.Map || tp.Kind() == reflect.Struct {
			t, _ = json.MarshalToString(val)
		} else {
			err = fmt.Errorf("%v transfer type to string error", val)
		}
	}
	return t, err
}

func ToBool(val any) (bool, error) {
	var t bool
	var err error

	switch val.(type) {
	case uint8, uint16, uint32, uint, uint64, int8, int16, int32, int, int64, float64, float32:
		intVal, _ := ToInt(val)
		t = intVal == 1
	case string:
		t, err = strconv.ParseBool(val.(string))
	default:
		err = fmt.Errorf("%v transfer type to bool error", val)
	}
	return t, err
}

func ToSlice(val any) ([]any, error) {
	var t []any
	var err error

	switch val.(type) {
	case []any:
		t = val.([]any)
	case string:
		if json.Unmarshal([]byte(val.(string)), &t) != nil {
			err = fmt.Errorf("%v transfer type to slice error", val)
		}
	default:
		tp := reflect.TypeOf(val)
		if tp.Kind() == reflect.Slice {
			byteData, _ := json.Marshal(val)
			_ = json.Unmarshal(byteData, &t)
		} else {
			err = fmt.Errorf("%v transfer type to slice error", val)
		}
	}
	return t, err
}

func ToMap(val any) (map[string]any, error) {
	var t = make(map[string]any)
	var err error

	switch val.(type) {
	case map[string]any:
		t = val.(map[string]any)
	case string:
		if json.Unmarshal([]byte(val.(string)), &t) != nil {
			err = fmt.Errorf("%v transfer type to map error", val)
		}
	default:
		tp := reflect.TypeOf(val)
		if tp.Kind() == reflect.Map {
			byteData, _ := json.Marshal(val)
			_ = json.Unmarshal(byteData, &t)
		} else {
			err = fmt.Errorf("%v transfer type to slice error", val)
		}
	}
	return t, err
}
