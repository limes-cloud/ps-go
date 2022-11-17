package engine

import (
	"context"
	"crypto/md5"
	"fmt"
	json "github.com/json-iterator/go"
	"ps-go/consts"
	"ps-go/errors"
	"time"
)

type runCache struct {
	*runtime
}

type cacheData struct {
	Data any `json:"data"`
}

func (r *runCache) cacheKey() string {
	byteData, _ := json.Marshal(r.component)
	return fmt.Sprintf("runtime_%x", md5.Sum(byteData))
}

func (r *runCache) getCache() (any, error) {
	key := r.cacheKey()
	str, err := r.ctx.Redis(consts.ProcessScheduleCache).Get(context.TODO(), key).Result()
	if err != nil || str == "" {
		return nil, errors.New("获取缓存失败")
	}
	cache := cacheData{}
	if json.UnmarshalFromString(str, &cache) != nil {
		return nil, errors.New("缓存数据解析失败")
	}
	return cache.Data, nil
}

func (r *runCache) setCache(value any) {
	key := r.cacheKey()
	data := cacheData{
		Data: value,
	}
	str, _ := json.MarshalToString(data)
	r.ctx.Redis(consts.ProcessScheduleCache).Set(context.TODO(), key, str, 5*time.Minute)
}
