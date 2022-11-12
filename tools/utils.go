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
