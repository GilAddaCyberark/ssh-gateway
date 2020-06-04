package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
)

// todo : move to config class

type ConfigGlobal struct {
	Server ServerConfig
	Dialer DialerConfig
}

var globalConfig *ConfigGlobal
var serverConfig *ServerConfig
var dialerConfig *DialerConfig

var configFilePath *string
var listeningPort *int
var enableRecording *bool
var recordingDir *string

func init() {
	// Init Flags
	setCommandLineArgs()

}
func main() {
	// Handle Command Line Args

	flag.Parse()

	printRuntimeArgs()

	// Load Configuration
	setConfig()

	// Set new SSH Server to listen to new incoming connections
	s := SSHGateway{}
	s.listeningPort = *listeningPort
	s.TargetInfo = &TargetInfo{}
	s.RelayInfo = &RelayInfo{}
	s.RelayInfo.EnableRecording = *enableRecording
	s.RelayInfo.RecordingsDir = *recordingDir

	if s.NewSSHGateway() != nil {
		panic("SSH Gatweay could not start")
	}
	s.ListenAndServe()
}
func setConfig() {
	// load configuration
	config, err := loadConfig()
	if err != nil {
		fmt.Printf("Load Configuration error: %v\n", err)
		panic(err)
	}

	serverConfig = &config.Server
	dialerConfig = &config.Dialer
	// relayConfig = &config.Relay
}

func loadConfig() (*ConfigGlobal, error) {
	// Print File
	fmt.Println(configFilePath)

	// todo: Check if files exists
	if fileExists(*configFilePath) {
		// Read File
		configData, err := ioutil.ReadFile(*configFilePath)
		if err != nil {
			return nil, fmt.Errorf("Failed to open config file: %s", err)
		}
		// Unmarshal json
		config := &ConfigGlobal{}
		err = json.Unmarshal(configData, config)
		if err != nil {
			return nil, fmt.Errorf("Unable to unmarhal json config file: %s", err)
		}
		return config, nil

	} else {
		return nil, fmt.Errorf("Unexpected error with json config file: %s", configFilePath)
	}
	return nil, fmt.Errorf("Unexpected error with json config file: %s", configFilePath)
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// *********************************
// Command Line Stuff
// *********************************
func printRuntimeArgs() {
	fmt.Println("SSH Gateway Running")
	fmt.Printf("Configuration File: %s\n", *configFilePath)
	fmt.Printf("SSH Gateway Listening Port: %d\n", *listeningPort)
	fmt.Printf("Session Recording Enabled: %s\n", *enableRecording)
	fmt.Printf("Session Recording Dir: %s\n", *recordingDir)
}

func setCommandLineArgs() {

	const (
		defaultConfigurtionFile = "config.json"
		defaultEnableRecording  = false
		defaultListentingPort   = 2222
		defaultRecordingDir     = "recordings"
	)

	// Command Line variabls
	const (
		logo = "EVEREST SSH BASTION Usage:\n" +
			"---------------------------\n"
	)

	configFilePath = flag.String("cfg", defaultConfigurtionFile, "The path of the ssh-gateway configuration file")
	listeningPort = flag.Int("port", defaultListentingPort, "The port that the ssh gateway is listening to client connections")
	recordingDir = flag.String("rec-path", defaultRecordingDir, "The path of the session recording dir")
	enableRecording = flag.Bool("rec", defaultEnableRecording, "To enable the recording of the client sessions. The value is true / false")
	flag.Usage = func() {
		fmt.Printf(logo)
		fmt.Printf("Usage: ssh-gateway [options] param>\n\n")
		flag.PrintDefaults()
	}
}
