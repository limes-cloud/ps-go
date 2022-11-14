package service

import (
	"github.com/jinzhu/copier"
	"github.com/limeschool/gin"
	"ps-go/errors"
	"ps-go/model"
	"ps-go/types"
)

func GetSecret(ctx *gin.Context, in *types.GetSecretRequest) (model.Secret, error) {
	var err error
	Secret := model.Secret{}
	if in.ID != 0 {
		err = Secret.OneByID(ctx, in.ID)
	}

	if in.Name != "" {
		err = Secret.OneByName(ctx, in.Name)
	}

	return Secret, err
}

func PageSecret(ctx *gin.Context, in *types.PageSecretRequest) ([]model.Secret, int64, error) {
	Secret := model.Secret{}
	return Secret.Page(ctx, in.Page, in.Count, in)
}

func AddSecret(ctx *gin.Context, in *types.AddSecretRequest) error {
	Secret := model.Secret{}
	if copier.Copy(&Secret, in) != nil {
		return errors.AssignError
	}
	return Secret.Create(ctx)
}

func UpdateSecret(ctx *gin.Context, in *types.UpdateSecretRequest) error {
	Secret := model.Secret{}
	if copier.Copy(&Secret, in) != nil {
		return errors.AssignError
	}
	return Secret.Update(ctx)
}

func DeleteSecret(ctx *gin.Context, in *types.DeleteSecretRequest) error {
	Secret := model.Secret{}
	if copier.Copy(&Secret, in) != nil {
		return errors.AssignError
	}
	return Secret.DeleteByID(ctx)
}
