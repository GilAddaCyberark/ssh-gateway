package ssh_engine

import (
	cfg "ssh-gateway/configs"
	cache "ssh-gateway/internal/pkg/ssh-engine/cache"
	"sync"
	"time"
)

var IDLE_TIMEOUT_SEC int = -1
var MAX_SESSION_DURATION_SEC int = -1

type SessionManager struct {
	OpenSessions sync.Map
}

func NewSessionManager() *SessionManager {
	sm := SessionManager{}
	return &sm
}

func (sm *SessionManager) StartTerminationThread() {
	IDLE_TIMEOUT_SEC = cfg.DataFlow_Config.IdleSessionTimeoutSec
	MAX_SESSION_DURATION_SEC = cfg.DataFlow_Config.MaxSessionDurationSec

	if IDLE_TIMEOUT_SEC <= 0 && MAX_SESSION_DURATION_SEC <= 0 {
		return
	}

	go func() {
		for range time.Tick(time.Second) {
			sm.OpenSessions.Range(func(key, value interface{}) bool {
				relay := value.(*SSHRelay)
				if relay != nil && relay.Controller != nil {
					if (IDLE_TIMEOUT_SEC > 0 && !cache.IsDelayLessThan(relay.Controller.LastUserInputTime, time.Now(), IDLE_TIMEOUT_SEC)) ||
						(MAX_SESSION_DURATION_SEC > 0 && !cache.IsDelayLessThan(relay.Controller.LastUserInputTime, time.Now(), IDLE_TIMEOUT_SEC)) {
						_ = relay.Controller.SendMessageToUser("$$$$$$$$$$$$$$$      SESSION DISCONNECTED    $$$$$$$$$$$$$$$$$\r\n")
						_ = relay.Controller.TerminateSession()
					}
				}
				return true
			})
		}
	}()
}

func (sm *SessionManager) AddNewSession(ssh *SSHRelay) {
	sm.OpenSessions.Store(ssh.RelayTargetInfo.SessionId, ssh)
}

func (sm *SessionManager) RemoveSession(ssh *SSHRelay) {
	sm.OpenSessions.Delete(ssh.RelayTargetInfo.SessionId)
}

func (sm *SessionManager) GetOpenSessionsCount() int {
	length := 0
	sm.OpenSessions.Range(func(_, _ interface{}) bool {
		length++
		return true
	})
	return length
}
