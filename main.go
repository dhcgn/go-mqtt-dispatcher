package main

import (
	"flag"
	"fmt"
	"go-mqtt-dispatcher/config"
	"go-mqtt-dispatcher/dispatcher"
	"log"
	"net/url"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
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
	configPathFlag = flag.String("config", "", "config file path, e.g. config.yaml")
	configCheck    = flag.Bool("config-check", false, "check config file and exit")
)

func main() {
	fmt.Printf("%s %s (commit: %s, built at: %s)\n", AppName, Version, Commit, BuildTime)
	fmt.Println("Url: https://github.com/dhcgn/go-mqtt-dispatcher")
	fmt.Println()

	flag.Parse()
	if *configPathFlag == "" {
		flag.PrintDefaults()
		return
	}

	// Load config
	config, err := config.LoadConfig(*configPathFlag)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if *configCheck {
		log.Println("Config check successful")
		return
	}

	// Create MQTT client
	client, err := connect(AppName+Version+Commit+BuildTime, config.Mqtt.BrokerAsUri, config.Mqtt.Username, config.Mqtt.Password)
	if err != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", err)
	}
	mqttClient := dispatcher.NewPahoMqttClient(client)

	d, err := dispatcher.NewDispatcher(&config.DispatcherEntries, mqttClient, func(s string) { log.Println("Disp: " + s) })
	if err != nil {
		log.Fatalf("Failed to create dispatcher: %v", err)
	}
	d.Run()

	select {}
}

func connect(clientId string, uri *url.URL, username, password string) (mqtt.Client, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s", uri.Host))
	if username != "" {
		opts.SetUsername(username)

	}
	if password != "" {
		opts.SetPassword(password)
	}

	opts.SetClientID(clientId)

	client := mqtt.NewClient(opts)
	token := client.Connect()
	for !token.WaitTimeout(3 * time.Second) {
	}
	if err := token.Error(); err != nil {
		return nil, err
	}
	return client, nil
}
