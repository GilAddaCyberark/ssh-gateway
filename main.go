package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
)

var configFilename *string

type ConfigGlobal struct {
	Server ServerConfig
	Dialer DialerConfig
	Relay  RelayConfig
}

var globalConfig *ConfigGlobal
var bastionConfig *ServerConfig
var dialerConfig *DialerConfig
var relayConfig *RelayConfig

func init() {
	// Set Flags
	configFilename = flag.String("config", "", "Configuration file in json")
	flag.Usage = printUsage
}
func main() {
	flag.Parse()
	// load configuration
	config, err := loadConfig()
	bastionConfig = &config.Server
	dialerConfig = &config.Dialer
	relayConfig = &config.Relay
	if err != nil {
		panic(err)
	}

	// Set new SSH Server to listen to new incoming connections
	s := SSHGateway{}
	if s.NewSSHGateway() != nil {
		panic(err)
	}
	s.ListenAndServe(bastionConfig.ServerAddress)
}

func printUsage() {
	fmt.Printf("Usage: yourtool [options] param>\n\n")
	flag.PrintDefaults()
}

func loadConfig() (*ConfigGlobal, error) {
	// Print File
	fmt.Println(*configFilename)

	// todo: Check if files exists
	if fileExists(*configFilename) {
		// Read File
		configData, err := ioutil.ReadFile(*configFilename)
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
		return nil, fmt.Errorf("Unexpected error with json config file: %s", *configFilename)
	}
	return nil, fmt.Errorf("Unexpected error with json config file: %s", *configFilename)
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
