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

var scriptKey = "script_lock"

type Script struct {
	Name       string `json:"name"`
	Script     string `json:"script,omitempty"`
	Version    string `json:"version"`
	Status     *bool  `json:"status"`
	Operator   string `json:"operator,omitempty"`
	OperatorID int64  `json:"operator_id,omitempty"`
	gin.DeleteModel
}

func (u Script) Table() string {
	return "script"
}

// Page 查询分页规则数据
func (u *Script) Page(ctx *gin.Context, page, count int, m interface{}, fs ...callback) ([]Script, int64, error) {
	var list []Script
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
func (u *Script) Count(ctx *gin.Context, fs ...callback) (int64, error) {
	var total int64

	db := database(ctx).Table(u.Table())
	db = exec(db, fs...)

	if err := db.Where("deleted_at is null").Count(&total).Error; err != nil {
		return total, err
	}
	return total, nil
}

func (u *Script) CacheKey(key string) string {
	return fmt.Sprintf("script_%v", key)
}

// OneByCache 通过key查询缓存
func (u *Script) OneByCache(ctx *gin.Context, key string) (bool, error) {
	byteData, err := cache(ctx).Get(ctx, key).Bytes()
	if err != nil {
		return false, err
	}

	if err = json.Unmarshal(byteData, u); err != nil {
		return false, err
	}

	if u.ID == 0 {
		return false, gorm.ErrRecordNotFound
	}

	return true, nil
}

// OneByName 通过name查询脚本
func (u *Script) OneByName(ctx *gin.Context, name string) error {
	if is, err := u.OneByCache(ctx, u.CacheKey(name)); is {
		return err
	}

	// 加锁,防止缓存击穿
	rl := lock.NewLock(ctx, scriptKey)
	rl.Acquire()
	defer rl.Release()

	// 获取锁之后重新查询缓存
	if is, err := u.OneByCache(ctx, u.CacheKey(name)); is {
		return err
	}

	db := database(ctx).Table(u.Table())
	if err := db.Where("status=true").Where("name=?", name).First(u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			cache(ctx).Set(ctx, u.CacheKey(name), "{}", time.Minute*5)
		}
		return err
	}

	str, _ := json.MarshalToString(u)
	cache(ctx).Set(ctx, u.CacheKey(name), str, 24*time.Hour)

	return nil
}

// OneByVersion 通过version查询脚本
func (u *Script) OneByVersion(ctx *gin.Context, version string) error {
	if is, err := u.OneByCache(ctx, u.CacheKey(version)); is {
		return err
	}

	// 加锁,防止缓存击穿
	rl := lock.NewLock(ctx, scriptKey)
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
func (u *Script) OneByID(ctx *gin.Context, id int64) error {
	if err := database(ctx).Table(u.Table()).Where("id = ?", id).First(u).Error; err != nil {
		return err
	}
	return nil
}

// Create 创建脚本
func (u *Script) Create(ctx *gin.Context) error {
	db := database(ctx)
	// 查看当前是否存在脚本
	count, _ := u.Count(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Table(u.Table()).Where("name = ?", u.Name)
	})

	// 创建脚本,第一个脚本则直接使用
	u.Status = tools.Bool(count == 0)
	u.Version = tools.UUID()
	if err := db.Table(u.Table()).Create(u).Error; err != nil {
		return err
	}

	// 判断是否超过保存最大的副本数量
	if count > consts.ScriptHistoryCount {
		script := Script{}
		if err := db.Table(u.Table()).
			Where("name = ?", u.Name).
			Order("id desc").
			Offset(consts.ScriptHistoryCount - 1).
			Limit(1).
			First(&script).Error; err == nil {

			db.Table(u.Table()).
				Where("id <= ? and name = ? and status = false", script.ID, script.Name).
				Delete(&Script{})

		}
	}

	return nil
}

// SwitchVersion 切换使用版本
func (u *Script) SwitchVersion(ctx *gin.Context) error {

	if err := u.OneByID(ctx, u.ID); err != nil {
		return err
	}

	if *u.Status == true {
		return nil
	}

	db := database(ctx).Table(u.Table())
	// 延迟双删
	delayDelCache(ctx, u.CacheKey(u.Name))

	// 进行版本切换，使用指定id版本
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("name=", u.Name).Update("status", false).Error; err != nil {
			return err
		}

		u.Status = tools.Bool(true)
		return tx.Where("id = ?", u.ID).Update("status", true).Error
	})
}

// DeleteByID 通过id删除规则
func (u *Script) DeleteByID(ctx *gin.Context) error {
	if err := u.OneByID(ctx, u.ID); err != nil {
		return err
	}

	if *u.Status {
		return errors.New("启用中的脚本不允许删除")
	}

	db := database(ctx).Table(u.Table())
	if err := db.Updates(u).Delete(u).Error; err != nil {
		return err
	}
	return nil
}
