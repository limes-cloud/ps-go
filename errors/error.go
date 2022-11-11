package errors

import (
	"errors"
	"fmt"
	"github.com/limeschool/gin"
)

const (
	DefaultCode = 100001 //通用错误吗
)

var (
	New = func(msg string) error {
		return &gin.CustomError{
			Code: DefaultCode,
			Msg:  msg,
		}
	}

	NewF = func(msg string, arg ...interface{}) error {
		return &gin.CustomError{
			Code: DefaultCode,
			Msg:  fmt.Sprintf(msg, arg...),
		}
	}

	Is = func(err, tar error) bool {
		return errors.Is(err, tar)
	}
)
