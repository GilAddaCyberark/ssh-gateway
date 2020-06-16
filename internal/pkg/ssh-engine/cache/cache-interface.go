package cache

import (
	"sync"
	"time"
)

type CacheInterface interface {
	CheckCacheItemValidity(time_created time.Time, valid_period_min int) bool
	GetItemsCount() int
}

type BaseCache struct {
	CacheInterface
	CacheMap sync.Map
}

func (*BaseCache) CheckCacheItemValidity(time_created time.Time, valid_period_min int) bool {
	cur_time := time.Now()
	return IsDelayLessThan(time_created, cur_time, valid_period_min*60)
}

func (bc *BaseCache) GetItemsCount() int {
	length := 0
	bc.CacheMap.Range(func(_, _ interface{}) bool {
		length++
		return true
	})
	return length
}

func IsDelayLessThan(t1 time.Time, t2 time.Time, delay_sec int) bool {
	min_diff := t2.Sub(t1).Seconds()
	retV := float64(delay_sec)-min_diff > 0
	return retV
}
