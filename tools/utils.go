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
