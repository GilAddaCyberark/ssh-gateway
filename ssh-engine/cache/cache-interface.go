package cache

import "time"

type CacheInterface interface {
	CheckCacheItemValidity(time_created time.Time, valid_period_min int) bool
}

type BaseCache struct {
	CacheInterface
}

func (BaseCache) CheckCacheItemValidity(time_created time.Time, valid_period_min int) bool {
	cur_time := time.Now()
	min_diff := cur_time.Sub(time_created).Minutes()
	retV := float64(valid_period_min)-min_diff > 0
	return retV
}
