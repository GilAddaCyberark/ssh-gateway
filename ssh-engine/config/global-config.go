package config

type ConfigGlobal struct {
	Server ServerConfig
	Dialer DialerConfig
	AWS    AWSConfig
}

type ServerConfig struct {
	ServerAddress       string
	ServerKeyPath       string
	User                string
	PrivateKeyPath      string
	PublicKeyPath       string
	ExpirationPeriodSec int
	// CertificateTokenIdTemplate string
}

type AWSConfig struct {
	TenantId           string
	PhysicalLambdaName string
	DefaultRegion      string
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
var ConfigFilePath *string
var ListeningPort *int

// var Session_Manager SessionManager = *NewSessionManager()
