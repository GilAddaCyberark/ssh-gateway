package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	rec "ssh-gateway/recorders"
	eng "ssh-gateway/ssh-engine"
	gen "ssh-gateway/ssh-engine/generic-structs"
)

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
	s := eng.SSHGateway{}
	s.ListeningPort = *eng.ListeningPort
	s.TargetInfo = &gen.TargetInfo{}
	s.RelayInfo = &eng.RelayInfo{}
	s.RelayInfo.EnableRecording = *rec.EnableRecording
	s.RelayInfo.RecordingsDir = *rec.RecordingDir

	if s.NewSSHGateway() != nil {
		panic("SSH Gatweay could not start")
	}

	err := s.ListenAndServe()
	if err != nil {
		fmt.Printf("ListenAndServe error: %v\n", err)
		return
	}

}
func setConfig() {
	// load configuration
	config, err := loadConfig()
	if err != nil {
		fmt.Printf("Load Configuration error: %v\n", err)
		os.Exit(eng.FileNotFoundExitCode)
	}

	eng.Server_Config = &config.Server
	eng.Dialer_Config = &config.Dialer
	// relayConfig = &config.Relay
}

func loadConfig() (*eng.ConfigGlobal, error) {
	// Print File
	fmt.Println(eng.ConfigFilePath)

	// todo: Check if files exists
	if fileExists(*eng.ConfigFilePath) {
		// Read File
		configData, err := ioutil.ReadFile(*eng.ConfigFilePath)
		if err != nil {
			return nil, fmt.Errorf("Failed to open config file: %s", err)
		}
		// Unmarshal json
		config := &eng.ConfigGlobal{}
		err = json.Unmarshal(configData, config)
		if err != nil {
			return nil, fmt.Errorf("Unable to unmarhal json config file: %s", err)
		}
		return config, nil

	}

	err := fmt.Errorf("Config file not found: %s", *eng.ConfigFilePath)
	return nil, err
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
	fmt.Printf("Configuration File: %s\n", *eng.ConfigFilePath)
	fmt.Printf("SSH Gateway Listening Port: %d\n", *eng.ListeningPort)
	fmt.Printf("Session Recording Enabled: %t\n", *rec.EnableRecording)
	fmt.Printf("Session Recording Dir: %s\n", *rec.RecordingDir)
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

	eng.ConfigFilePath = flag.String("cfg", defaultConfigurtionFile, "The path of the ssh-gateway configuration file")
	eng.ListeningPort = flag.Int("port", defaultListentingPort, "The port that the ssh gateway is listening to client connections")
	rec.RecordingDir = flag.String("rec-path", defaultRecordingDir, "The path of the session recording dir")
	rec.EnableRecording = flag.Bool("rec", defaultEnableRecording, "To enable the recording of the client sessions. The value is true / false")
	flag.Usage = func() {
		fmt.Printf(logo)
		fmt.Printf("Usage: ssh-gateway [options] param>\n\n")
		flag.PrintDefaults()
	}
}
