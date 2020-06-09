package ssh_engine

import "sync"

type SessionManager struct {
	OpenSessions sync.Map
}

func NewSessionManager() *SessionManager {
	sm := SessionManager{}
	return &sm
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
