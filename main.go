package main

import (
	"flag"
	"fmt"
	"go-mqtt-dispatcher/dispatcher"
	"go-mqtt-dispatcher/types"
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
	config, err := types.LoadConfig(*confgFlag)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create MQTT client
	client := connect(AppName+Version+Commit+BuildTime, config.Mqtt.BrokerAsUri)
	mqttClient := dispatcher.NewPahoMqttClient(client)

	d, err := dispatcher.NewDispatcher(&config.DispatcherEntries, mqttClient, func(s string) { log.Println("Disp: " + s) })
	if err != nil {
		log.Fatalf("Failed to create dispatcher: %v", err)
	}
	d.Run()

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
