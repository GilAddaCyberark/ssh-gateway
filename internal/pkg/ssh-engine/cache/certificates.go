package cache

import (
	"sync"
	"time"
)

const CERTIFICATE_TIMEOUT_MIN int = 5

type CertificateEntry struct {
	Certificate string
	CreateTime  time.Time
}

type CertificatesCacheManager struct {
	BaseCache
	Certificates sync.Map
}

func NewCertificatesCacheManager() *CertificatesCacheManager {
	sm := CertificatesCacheManager{}
	return &sm
}

func (ip *CertificatesCacheManager) GetCertificate(cert_key string) string {
	tip, ok := ip.Certificates.Load(cert_key)
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
	ip.Certificates.Delete(cert_key)
	ip.Certificates.Store(cert_key, ce)
}

func (ip *CertificatesCacheManager) GetItemsCount() int {
	length := 0
	ip.Certificates.Range(func(_, _ interface{}) bool {
		length++
		return true
	})
	return length
}
