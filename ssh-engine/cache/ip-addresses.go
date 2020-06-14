package cache

import (
	"sync"
	"time"
)

var PUBLIC_IP_TIMEOUT_MIN int = 5

type TargetIpEntry struct {
	TargetId   string
	PublicIP   string
	CreateTime time.Time
}

type PublicIpCacheManager struct {
	BaseCache
	IpAddresses sync.Map
}

func NewPublicIpCacheManager() *PublicIpCacheManager {
	sm := PublicIpCacheManager{}
	return &sm
}

func (ip *PublicIpCacheManager) GetPublicIp(target_id string) string {
	tip, ok := ip.IpAddresses.Load(target_id)
	if ok {
		// Public IP found in cache. Now check the time expiration.
		if ip.CheckCacheItemValidity(tip.(TargetIpEntry).CreateTime, PUBLIC_IP_TIMEOUT_MIN) {
			return tip.(TargetIpEntry).PublicIP
		}
	}
	return ""
}

func (ip *PublicIpCacheManager) SavePublicIP(target_id string, public_ip string) {
	tar := TargetIpEntry{target_id, public_ip, time.Now()}
	ip.IpAddresses.Delete(target_id)
	ip.IpAddresses.Store(target_id, tar)
}

func (ip *PublicIpCacheManager) GetItemsCount() int {
	length := 0
	ip.IpAddresses.Range(func(_, _ interface{}) bool {
		length++
		return true
	})
	return length
}
