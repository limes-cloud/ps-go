package handler

import (
	"github.com/limeschool/gin"
	"gorm.io/gorm"
	"ps-go/errors"
)

func TransferError(e error) error {
	if err, ok := e.(*gin.CustomError); ok {
		return err
	}
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return errors.DBNotFoundError
	}
	return errors.DBError
}
