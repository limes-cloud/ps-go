package model

import (
	"github.com/limeschool/gin"
	"gorm.io/gorm"
	"ps-go/errors"
	"ps-go/tools"
	"ps-go/tools/lock"
	"time"
)

var ruleLockKey = "lock_rule"

type Rule struct {
	ID         int64  `gorm:"primary_key" json:"id"`
	Name       string `json:"name"`
	Rule       string `json:"rule,omitempty"`
	Operator   string `json:"operator,omitempty"`
	OperatorID int64  `json:"operator_id,omitempty"`
	DeleteModel
}

func (u Rule) Table() string {
	return "rule"
}

// Page 查询分页规则数据
func (u *Rule) Page(ctx *gin.Context, page, count int, m interface{}, fs ...callback) ([]Rule, int64, error) {
	var list []Rule
	var total int64

	db := database(ctx).Table(u.Table()).Select("id,name,operator,operator_id,created_at,updated_at")
	db = gin.GormWhere(db, u.Table(), m).Where("deleted_at is null")
	db = exec(db, fs...)

	if err := db.Count(&total).Error; err != nil {
		return nil, total, err
	}

	if err := db.Order("created_at asc").Offset((page - 1) * count).Limit(count).Find(&list).Error; err != nil {
		return list, total, errors.DBError
	}

	return list, total, nil
}

// Count 查询分页规则数量
func (u *Rule) Count(ctx *gin.Context, fs ...callback) (int64, error) {
	var total int64

	db := database(ctx).Table(u.Table()).Where("deleted_at is null")
	db = exec(db, fs...)

	if err := db.Count(&total).Error; err != nil {
		return total, transferErr(err)
	}
	return total, nil
}

func (u *Rule) OneByNameCache(ctx *gin.Context, name string) (bool, error) {
	byteData, err := cache(ctx).Get(ctx, name).Bytes()
	if err != nil {
		return false, err
	}

	if err = json.Unmarshal(byteData, u); err != nil {
		return false, err
	}

	if u.ID == 0 {
		return true, gorm.ErrRecordNotFound
	}

	return false, nil
}

// OneByName 通过name查询规则详情（带缓存）
func (u *Rule) OneByName(ctx *gin.Context, name string) error {
	if is, err := u.OneByNameCache(ctx, name); is {
		return err
	}

	// 加锁,防止缓存击穿
	rl := lock.NewLock(ctx, ruleLockKey)
	rl.Acquire()
	defer rl.Release()

	// 获取锁之后重新查redis
	if is, err := u.OneByNameCache(ctx, name); is {
		return err
	}

	db := database(ctx).Table(u.Table())
	if err := db.Where("deleted_at is null").Where("name = ?", name).First(u).Error; err != nil {
		// 防止缓存穿透
		if errors.Is(err, gorm.ErrRecordNotFound) {
			cache(ctx).Set(ctx, name, "{}", time.Second*5)
		}
		return transferErr(err)
	}

	str, _ := json.MarshalToString(u)
	cache(ctx).Set(ctx, name, str, 24*time.Hour)

	return nil
}

// OneByID 通过id查询规则详情
func (u *Rule) OneByID(ctx *gin.Context, id int64) error {
	err := database(ctx).Table(u.Table()).Where("deleted_at is null").Where("id = ?", id).First(u).Error
	if err != nil {
		return transferErr(err)
	}
	return nil
}

// Create 创建规则
func (u *Rule) Create(ctx *gin.Context) error {

	count, _ := u.Count(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ?", u.Name)
	})
	if count != 0 {
		return errors.New("规则名称已存在")
	}

	delayDelCache(ctx, u.Name)

	if err := database(ctx).Table(u.Table()).Create(u).Error; err != nil {
		return transferErr(err)
	}

	return nil
}

// UpdateByID 通过Name删除规则（删除缓存）
func (u *Rule) UpdateByID(ctx *gin.Context) error {
	rule := Rule{}
	if err := rule.OneByID(ctx, u.ID); err != nil {
		return err
	}

	tempRule := Rule{}
	if err := tempRule.OneByName(ctx, u.Name); !errors.Is(err, gorm.ErrRecordNotFound) && tempRule.ID != rule.ID {
		return errors.New("规则名称已存在")
	}

	// 延迟双删
	delayDelCache(ctx, rule.Name)

	if database(ctx).Table(u.Table()).Where("id = ?", u.ID).Updates(u).Error != nil {
		return errors.DBError
	}

	return nil
}

// DeleteByName 通过Name删除规则（删除缓存）
func (u *Rule) DeleteByName(ctx *gin.Context, name string) error {
	db := database(ctx).Table(u.Table())
	delayDelCache(ctx, name)

	u.DeletedAt = tools.Int64(time.Now().Unix())
	if err := db.Where("deleted_at is null").Where("name = ?", name).
		Updates(u).Error; err != nil {
		return errors.DBError
	}

	return nil
}
