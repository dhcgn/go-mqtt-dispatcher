package main

import (
	"flag"
	"fmt"
	"go-mqtt-dispatcher/dispatcher"
	"go-mqtt-dispatcher/types"
	"log"
	"net/url"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
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
	confgFlag = flag.String("config", "", "config file path, e.g. config.yaml")
)

func main() {
	fmt.Printf("%s %s (commit: %s, built at: %s)\n", AppName, Version, Commit, BuildTime)
	fmt.Println("Url: https://github.com/dhcgn/go-mqtt-dispatcher")
	fmt.Println()

	flag.Parse()
	if *confgFlag == "" {
		flag.PrintDefaults()
		return
	}

	// Load config
	config, err := LoadConfig(*confgFlag)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create MQTT client
	client := connect(AppName, config.Mqtt.BrokerAsUri)
	mqttClient := dispatcher.NewPahoMqttClient(client)

	// Create dispatcher with MQTT client
	d, err := dispatcher.NewDispatcher(config, mqttClient)
	if err != nil {
		log.Fatalf("Failed to create dispatcher: %v", err)
	}

	log.Println("Start dispatcher")
	defer log.Println("Done")

	go d.Run(AppName, Version, Commit, BuildTime)

	select {}
}

func connect(clientId string, uri *url.URL) mqtt.Client {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s", uri.Host))
	// opts.SetUsername(uri.User.Username())
	// password, _ := uri.User.Password()
	// opts.SetPassword(password)
	opts.SetClientID(clientId)

	client := mqtt.NewClient(opts)
	token := client.Connect()
	for !token.WaitTimeout(3 * time.Second) {
	}
	if err := token.Error(); err != nil {
		log.Fatal(err)
	}
	return client
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
