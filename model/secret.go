package model

import (
	"encoding/base64"
	"fmt"
	"github.com/limeschool/gin"
	"gorm.io/gorm"
	"ps-go/consts"
	"ps-go/errors"
	"ps-go/tools/lock"
	"time"
)

var secretKey = "secret_lock"

type Secret struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Context     string `json:"context"`
	Operator    string `json:"operator,omitempty"`
	OperatorID  int64  `json:"operator_id,omitempty"`
	gin.DeleteModel
}

func (s Secret) Table() string {
	return "secret"
}

// Page 查询分页规则数据
func (s *Secret) Page(ctx *gin.Context, page, count int, m interface{}, fs ...callback) ([]Secret, int64, error) {
	var list []Secret
	var total int64
	db := database(ctx).Table(s.Table())
	db = db.Select("id,name,operator,operator_id,created_at,updated_at,description")
	db = gin.GormWhere(db, s.Table(), m)
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
func (s *Secret) Count(ctx *gin.Context, fs ...callback) (int64, error) {
	var total int64

	db := database(ctx).Table(s.Table())
	db = exec(db, fs...)

	if err := db.Where("deleted_at is null").Count(&total).Error; err != nil {
		return total, err
	}
	return total, nil
}

func (s *Secret) CacheKey(key string) string {
	return fmt.Sprintf("secret_%v", key)
}

// OneByCache 通过key查询缓存
func (s *Secret) OneByCache(ctx *gin.Context, key string) (bool, error) {
	byteData, err := cache(ctx).Get(ctx, key).Bytes()
	if err != nil {
		return false, err
	}

	if len(byteData) == 0 {
		return false, errors.DBNotFoundError
	}

	if err = json.Unmarshal(byteData, s); err != nil {
		return false, err
	}

	if s.ID == 0 {
		return true, gorm.ErrRecordNotFound
	}

	return true, nil
}

// OneByName 通过name查询脚本
func (s *Secret) OneByName(ctx *gin.Context, name string) error {
	if is, err := s.OneByCache(ctx, s.CacheKey(name)); is {
		return err
	}

	// 加锁,防止缓存击穿
	rl := lock.NewLock(ctx, secretKey)
	rl.Acquire()
	defer rl.Release()

	// 获取锁之后重新查询缓存
	if is, err := s.OneByCache(ctx, s.CacheKey(name)); is {
		return err
	}

	db := database(ctx).Table(s.Table())
	if err := db.Where("name=?", name).First(s).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			cache(ctx).Set(ctx, s.CacheKey(name), "{}", time.Minute*5)
		}
		return err
	}

	if byteData, err := base64.StdEncoding.DecodeString(s.Context); err != nil {
		return errors.New("密钥格式出错")
	} else {
		s.Context = string(byteData)
	}

	str, _ := json.MarshalToString(s)
	cache(ctx).Set(ctx, s.CacheKey(name), str, 24*time.Hour)

	return nil
}

// OneByID 通过id查询规则详情
func (s *Secret) OneByID(ctx *gin.Context, id int64) error {
	if err := database(ctx).Table(s.Table()).Where("id = ?", id).First(s).Error; err != nil {
		return err
	}

	if byteData, err := base64.StdEncoding.DecodeString(s.Context); err != nil {
		return errors.New("密钥格式出错")
	} else {
		s.Context = string(byteData)
	}

	return nil
}

// Create 创建脚本
func (s *Secret) Create(ctx *gin.Context) error {
	db := database(ctx).Table(s.Table())
	// 查看当前是否存在脚本
	count, _ := s.Count(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ?", s.Name)
	})

	if count != 0 {
		return errors.DBDupError
	}

	s.Context = base64.StdEncoding.EncodeToString([]byte(s.Context))

	// 不存在则直接创建
	return db.Create(s).Error
}

// Update 更新密钥信息
func (s *Secret) Update(ctx *gin.Context) error {
	// 判断修改的数据是否存在
	secret := Secret{}
	if err := secret.OneByID(ctx, s.ID); err != nil {
		return err
	}

	db := database(ctx).Table(s.Table()).Session(&gorm.Session{NewDB: true})

	// 判断是否修改名字
	if s.Name != secret.Name {
		var count int64
		db.Where("name=?", s.Name).Count(&count)
		if count != 0 {
			return errors.NewF("密钥名%v已存在", s.Name)
		}
		// 删除之前的缓存
		ctx.Redis(consts.ProcessScheduleCache).Del(ctx, s.CacheKey(secret.Name))
	}

	// 延迟双删
	delayDelCache(ctx, s.CacheKey(s.Name))

	if s.Context != "" {
		s.Context = base64.StdEncoding.EncodeToString([]byte(s.Context))
	}
	// 进行版本切换，使用指定id版本
	return db.Updates(s).Error
}

// DeleteByID 通过id删除规则
func (s *Secret) DeleteByID(ctx *gin.Context) error {
	if err := s.OneByID(ctx, s.ID); err != nil {
		return err
	}

	db := database(ctx).Table(s.Table())
	if err := db.Updates(s).Delete(s).Error; err != nil {
		return err
	}
	return nil
}
