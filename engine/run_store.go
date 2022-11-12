package engine

import (
	"regexp"
	"strings"
	"sync"
)

type RunStore interface {
	SetData(key string, val any)
	GetData(key string) any
	GetMatchData(m any) any
	GetAll() map[string]any
}

type runStore struct {
	data map[string]any
	lock sync.RWMutex
}

func (r *runStore) GetAll() map[string]any {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.data
}

// SetData 递归设置数据
func (r *runStore) SetData(key string, val any) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.data[key] = val
}

// GetData 直接通过viper 获取数据
func (r *runStore) GetData(key string) any {
	r.lock.RLock()
	defer r.lock.RUnlock()
	keys := strings.Split(key, ".")
	if len(keys) == 1 {
		return r.data[key]
	}

	var temp any = r.data
	// 遍历取值
	for _, key = range keys {
		temp = r.get(key, temp)
	}

	return temp
}

func (r *runStore) get(key string, data any) any {
	switch data.(type) {
	case map[string]any:
		return data.(map[string]any)[key]

	case map[string]uint:
		return data.(map[string]uint)[key]

	case map[string]int8:
		return data.(map[string]int8)[key]

	case map[string]uint8:
		return data.(map[string]uint8)[key]

	case map[string]int16:
		return data.(map[string]int16)[key]

	case map[string]uint16:
		return data.(map[string]uint16)[key]

	case map[string]int32:
		return data.(map[string]int32)[key]

	case map[string]uint32:
		return data.(map[string]uint32)[key]

	case map[string]int64:
		return data.(map[string]int64)[key]

	case map[string]uint64:
		return data.(map[string]uint64)[key]

	case map[string]float32:
		return data.(map[string]float32)[key]

	case map[string]float64:
		return data.(map[string]float32)[key]

	case map[string]string:
		return data.(map[string]string)[key]

	case map[string]int:
		return data.(map[string]int)[key]

	case map[string]bool:
		return data.(map[string]bool)[key]

	default:
		return nil
	}
}

// GetMatchData 获取存在表达式的数据
func (r *runStore) GetMatchData(m any) any {
	reg := regexp.MustCompile(`\{(\w|\.)+\}`)

	switch m.(type) {
	case []any:
		var resp = m.([]any)
		for key, _ := range resp {
			resp[key] = r.GetMatchData(resp[key])
		}
		return resp

	case string:
		if str := reg.FindString(m.(string)); str != "" {
			return r.GetData(str[1 : len(str)-1])
		}

	case map[string]any:
		var resp = m.(map[string]any)
		for key, _ := range resp {
			resp[key] = r.GetMatchData(resp[key])
		}
		return resp
	}

	return m
}
