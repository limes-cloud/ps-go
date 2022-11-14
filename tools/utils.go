package tools

import (
	"crypto/md5"
	"fmt"
	"github.com/google/uuid"
	"strings"
	"unsafe"
)

func UUID() string {
	uid := uuid.New().String()
	d := md5.Sum(*(*[]byte)(unsafe.Pointer(&uid)))
	return strings.ToUpper(fmt.Sprintf("%x", d))
}

type ListType interface {
	~string | ~int | ~int64 | ~[]byte | ~rune | ~float64
}

func InList[ListType comparable](list []ListType, val ListType) bool {
	for _, v := range list {
		if v == val {
			return true
		}
	}
	return false
}

// GetMapData 取map数据，带.
func GetMapData(key string, m map[string]any) any {
	keys := strings.Split(key, ".")
	if len(keys) == 1 {
		return m[key]
	}

	var temp any = m
	// 遍历取值
	for _, key = range keys {
		temp = getMapByAny(key, temp)
	}
	return temp
}

func getMapByAny(key string, data any) any {
	switch data.(type) {
	case map[string]any:
		return data.(map[string]any)[key]
	case map[string]string:
		return data.(map[string]string)[key]
	default:
		return nil
	}
}
