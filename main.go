package main

import (
	"flag"
	"fmt"
	"go-mqtt-dispatcher/dispatcher"
	"go-mqtt-dispatcher/types"
	"log"
	"net/url"
	"os"

	"gopkg.in/yaml.v2"
)

const (
	AppName = "go-mqtt-dispatcher"
)

var (
	Version   = "dev"
	Commit    = "none"
	BuildTime = "unknown"
)

var (
	confgFlag = flag.String("config", "config.yaml", "config file path")
)

func main() {
	fmt.Printf("%s %s (commit: %s, built at: %s)\n", AppName, Version, Commit, BuildTime)

	flag.Parse()
	if *confgFlag == "" {
		flag.PrintDefaults()
		return
	}

	// Load config
	config, err := LoadConfig(*confgFlag)
	if err != nil {
		fmt.Println(err)
		return
	}

	d, err := dispatcher.NewDispatcher(config)
	if err != nil {
		fmt.Println(err)
		return
	}

	log.Println("Start dispatcher")
	defer log.Println("Done")

	go d.Run()

	select {}
}

func LoadConfig(path string) (*types.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg types.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Parse mqtt broker as url
	cfg.Mqtt.BrokerAsUri, err = url.Parse(cfg.Mqtt.Broker)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
