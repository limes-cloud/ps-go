package model

import (
	"fmt"
	"github.com/limeschool/gin"
	"gorm.io/gorm"
	"ps-go/consts"
	"ps-go/errors"
	"ps-go/tools"
	"ps-go/tools/lock"
	"time"
)

var ruleKey = "rule_lock"

type Rule struct {
	Name       string `json:"name"`
	Rule       string `json:"rule,omitempty"`
	Version    string `json:"version"`
	Status     *bool  `json:"status"`
	Operator   string `json:"operator,omitempty"`
	OperatorID int64  `json:"operator_id,omitempty"`
	gin.DeleteModel
}

func (u Rule) Table() string {
	return "rule"
}

// Page 查询分页规则数据
func (u *Rule) Page(ctx *gin.Context, page, count int, m interface{}, fs ...callback) ([]Rule, int64, error) {
	var list []Rule
	var total int64

	db := database(ctx).Table(u.Table()).Select("id,name,operator,operator_id,created_at,updated_at,status,version")
	db = gin.GormWhere(db, u.Table(), m)
	db = exec(db, fs...)

	if err := db.Where("deleted_at is null").Count(&total).Error; err != nil {
		return nil, total, err
	}

	if err := db.Order("created_at desc").Offset((page - 1) * count).Limit(count).Find(&list).Error; err != nil {
		return list, total, err
	}

	return list, total, nil
}

// Count 查询指定条件的数量
func (u *Rule) Count(ctx *gin.Context, fs ...callback) (int64, error) {
	var total int64

	db := database(ctx).Table(u.Table())
	db = exec(db, fs...)

	if err := db.Where("deleted_at is null").Count(&total).Error; err != nil {
		return total, err
	}
	return total, nil
}

func (u *Rule) CacheKey(key string) string {
	return fmt.Sprintf("rule_%v", key)
}

// OneByNameCache 通过name查询缓存
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

// OneByName 通过name查询规则
func (u *Rule) OneByName(ctx *gin.Context, name string) error {
	if is, err := u.OneByNameCache(ctx, u.CacheKey(name)); is {
		return err
	}

	// 加锁,防止缓存击穿
	rl := lock.NewLock(ctx, ruleKey)
	rl.Acquire()
	defer rl.Release()

	// 获取锁之后重新查询缓存
	if is, err := u.OneByNameCache(ctx, u.CacheKey(name)); is {
		return err
	}

	db := database(ctx).Table(u.Table())
	if err := db.Where("status=true").Where("name = ?", name).First(u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			cache(ctx).Set(ctx, u.CacheKey(name), "{}", time.Minute*5)
		}
		return err
	}

	str, _ := json.MarshalToString(u)
	cache(ctx).Set(ctx, u.CacheKey(name), str, 24*time.Hour)

	return nil
}

// OneByVersionCache 通过version查询缓存
func (u *Rule) OneByVersionCache(ctx *gin.Context, version string) (bool, error) {
	byteData, err := cache(ctx).Get(ctx, version).Bytes()
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

// OneByVersion 通过version查询规则
func (u *Rule) OneByVersion(ctx *gin.Context, version string) error {
	if is, err := u.OneByNameCache(ctx, u.CacheKey(version)); is {
		return err
	}

	// 加锁,防止缓存击穿
	rl := lock.NewLock(ctx, ruleKey)
	rl.Acquire()
	defer rl.Release()

	// 获取锁之后重新查询缓存
	if is, err := u.OneByVersionCache(ctx, u.CacheKey(version)); is {
		return err
	}

	db := database(ctx).Table(u.Table())
	if err := db.Where("version = ?", version).First(u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			cache(ctx).Set(ctx, u.CacheKey(version), "{}", time.Minute*5)
		}
		return err
	}

	str, _ := json.MarshalToString(u)
	cache(ctx).Set(ctx, u.CacheKey(version), str, 24*time.Hour)

	return nil
}

// OneByID 通过id查询规则详情
func (u *Rule) OneByID(ctx *gin.Context, id int64) error {
	if err := database(ctx).Table(u.Table()).Where("id = ?", id).First(u).Error; err != nil {
		return err
	}
	return nil
}

// Create 创建规则
func (u *Rule) Create(ctx *gin.Context) error {
	db := database(ctx)
	// 查看当前是否存在规则
	count, _ := u.Count(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Table(u.Table()).Where("name = ?", u.Name)
	})

	// 创建规则,第一个规则则直接使用
	u.Status = tools.Bool(count == 0)
	u.Version = tools.UUID()
	if err := db.Table(u.Table()).Create(u).Error; err != nil {
		return err
	}

	// 判断是否超过保存最大的副本数量
	if count > consts.RuleHistoryCount {
		rule := Rule{}
		if err := db.Table(u.Table()).Order("id desc").Offset(consts.RuleHistoryCount - 1).Limit(1).First(&rule).Error; err == nil {
			db.Table(u.Table()).Where("id <= ? and status = false", rule.ID).Delete(&Rule{})
		} else {
			fmt.Println(err.Error())
		}
	}

	return nil
}

// SwitchVersion 切换使用版本
func (u *Rule) SwitchVersion(ctx *gin.Context) error {
	rule := Rule{}
	if err := rule.OneByID(ctx, u.ID); err != nil {
		return err
	}

	u.Status = tools.Bool(true)

	db := database(ctx).Table(u.Table())
	// 延迟双删
	delayDelCache(ctx, u.CacheKey(u.Name))

	// 进行版本切换，使用指定id版本
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Updates(u).Error; err != nil {
			return err
		}
		return tx.Where("id != ?", u.ID).Update("status", false).Error
	})
}

// DeleteByID 通过id删除规则
func (u *Rule) DeleteByID(ctx *gin.Context) error {
	if err := u.OneByID(ctx, u.ID); err != nil {
		return err
	}

	if *u.Status {
		return errors.New("启用中的规则不允许删除")
	}

	db := database(ctx).Table(u.Table())
	if err := db.Updates(u).Delete(u).Error; err != nil {
		return err
	}
	return nil
}
