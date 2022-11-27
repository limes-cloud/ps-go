package model

import (
	"fmt"
	"github.com/limeschool/gin"
	"gorm.io/gorm"
	"ps-go/consts"
	"ps-go/errors"
	"ps-go/tools"
	"ps-go/tools/lock"
	"strings"
	"time"
)

var ruleKey = "rule_lock"

type Rule struct {
	Name       string `json:"name"`
	Method     string `json:"method"`
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

	db := database(ctx).Table(u.Table()).
		Select("id,name,method,operator,operator_id,created_at,updated_at,status,version")

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

// OneByCache 通过key查询缓存
func (u *Rule) OneByCache(ctx *gin.Context, key string) (bool, error) {
	byteData, err := cache(ctx).Get(ctx, key).Bytes()
	if err != nil {
		return false, err
	}

	if len(byteData) == 0 {
		return false, errors.DBNotFoundError
	}

	if err = json.Unmarshal(byteData, u); err != nil {
		return false, err
	}

	if u.ID == 0 {
		return true, gorm.ErrRecordNotFound
	}

	return true, nil
}

// OneByNameMethod 通过name和method查询规则
func (u *Rule) OneByNameMethod(ctx *gin.Context, name, method string) error {
	method = strings.ToUpper(method)
	cacheKey := u.CacheKey(fmt.Sprintf("%v:%v", name, method))

	if is, err := u.OneByCache(ctx, cacheKey); is {
		return err
	}

	// 加锁,防止缓存击穿
	rl := lock.NewLock(ctx, ruleKey)
	rl.Acquire()
	defer rl.Release()

	// 获取锁之后重新查询缓存
	if is, err := u.OneByCache(ctx, cacheKey); is {
		return err
	}

	db := database(ctx).Table(u.Table())
	if err := db.Where("name=? and method=? and status=true", name, method).First(u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			cache(ctx).Set(ctx, cacheKey, "{}", time.Minute*5)
		}
		return err
	}

	str, _ := json.MarshalToString(u)
	cache(ctx).Set(ctx, cacheKey, str, 24*time.Hour)

	return nil
}

// OneByVersion 通过version查询规则
func (u *Rule) OneByVersion(ctx *gin.Context, version string) error {
	if is, err := u.OneByCache(ctx, u.CacheKey(version)); is {
		return err
	}

	// 加锁,防止缓存击穿
	rl := lock.NewLock(ctx, ruleKey)
	rl.Acquire()
	defer rl.Release()

	// 获取锁之后重新查询缓存
	if is, err := u.OneByCache(ctx, u.CacheKey(version)); is {
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
	u.Method = strings.ToUpper(u.Method)

	// 延迟双删
	cacheKey := u.CacheKey(fmt.Sprintf("%v:%v", u.Name, u.Method))
	delayDelCache(ctx, cacheKey)

	db := database(ctx).Table(u.Table()).Session(&gorm.Session{NewDB: true})
	// 查看当前是否存在规则
	count, _ := u.Count(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ? and method = ?", u.Name, u.Method)
	})

	// 创建规则,第一个规则则直接使用
	u.Status = tools.Bool(count == 0)
	u.Version = tools.UUID()
	if err := db.Create(u).Error; err != nil {
		return err
	}

	// 判断是否超过保存最大的副本数量
	if count >= consts.RuleHistoryCount {
		rule := Rule{}
		if err := db.Where("name=? and method=?", u.Name, u.Method).
			Order("id desc").Offset(consts.RuleHistoryCount - 1).Limit(1).First(&rule).Error; err == nil {
			db.Where("id<=? and name=? and method=? and status=false", rule.ID, u.Name, u.Method).Delete(&Rule{})
		}
	}
	return nil
}

// SwitchVersion 切换使用版本
func (u *Rule) SwitchVersion(ctx *gin.Context) error {
	if err := u.OneByID(ctx, u.ID); err != nil {
		return err
	}

	if *u.Status == true {
		return nil
	}

	db := database(ctx).Table(u.Table())

	// 延迟双删
	cacheKey := u.CacheKey(fmt.Sprintf("%v:%v", u.Name, u.Method))
	delayDelCache(ctx, cacheKey)

	// 进行版本切换，使用指定id版本
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("name=? and method=?", u.Name, u.Method).
			Update("status", false).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", u.ID).Update("status", true).Error
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
