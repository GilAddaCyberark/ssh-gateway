package config

type ConfigGlobal struct {
	Server   ServerConfig
	Dialer   DialerConfig
	AWS      AWSConfig
	DataFlow DataFlowConfig
}

type ServerConfig struct {
	ServerAddress       string
	ServerKeyPath       string
	User                string
	PrivateKeyPath      string
	PublicKeyPath       string
	ExpirationPeriodSec int
	EnableFileRecorder  bool
	EnableCWLRecorder   bool
}

type AWSConfig struct {
	TenantId           string
	PhysicalLambdaName string
	DefaultRegion      string
}

type DataFlowConfig struct {
	IdleSessionTimeoutSec int
	MaxSessionDurationSec int
}

type DialerConfig struct {
	DefaultPort          int
	MinPortToRotate      int
	MaxPortToRotate      int
	AuthType             string
	User                 string
	Password             string
	DefaultTargetAddress string
}

const (
	FileNotFoundExitCode int = 1
)

var Global_Config *ConfigGlobal
var Server_Config *ServerConfig
var Dialer_Config *DialerConfig
var AWS_Config *AWSConfig
var DataFlow_Config *DataFlowConfig
var ConfigFilePath *string
var ListeningPort *int
