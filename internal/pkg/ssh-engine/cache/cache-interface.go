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

func (BaseCache) CheckCacheItemValidity(time_created time.Time, valid_period_min int) bool {
	cur_time := time.Now()
	min_diff := cur_time.Sub(time_created).Minutes()
	retV := float64(valid_period_min)-min_diff > 0
	return retV
}

func (bc *BaseCache) GetItemsCount() int {
	length := 0
	bc.CacheMap.Range(func(_, _ interface{}) bool {
		length++
		return true
	})
	return length
}

