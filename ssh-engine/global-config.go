package ssh_engine

type ConfigGlobal struct {
	Server ServerConfig
	Dialer DialerConfig
}

const (
	FileNotFoundExitCode int = 1
)

var Global_Config *ConfigGlobal
var Server_Config *ServerConfig
var Dialer_Config *DialerConfig

var ConfigFilePath *string
var ListeningPort *int
