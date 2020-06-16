package cache

import (
	"time"
)

const CERTIFICATE_TIMEOUT_MIN int = 5

type CertificateEntry struct {
	Certificate string
	CreateTime  time.Time
}

type CertificatesCacheManager struct {
	BaseCache
}

func NewCertificatesCacheManager() *CertificatesCacheManager {
	sm := CertificatesCacheManager{}
	return &sm
}

func (ip *CertificatesCacheManager) GetCertificate(cert_key string) string {
	tip, ok := ip.CacheMap.Load(cert_key)
	if ok {
		// Certificate found in cache. Now check the time expiration.
		if ip.CheckCacheItemValidity(tip.(CertificateEntry).CreateTime, CERTIFICATE_TIMEOUT_MIN) {
			return tip.(CertificateEntry).Certificate
		}
	}
	return ""
}

func (ip *CertificatesCacheManager) SaveCertificate(cert_key string, certificate string) {
	ce := CertificateEntry{certificate, time.Now()}
	ip.CacheMap.Delete(cert_key)
	ip.CacheMap.Store(cert_key, ce)
}
