package main

import (
	"flag"
	"fmt"
	"go-mqtt-dispatcher/config"
	"go-mqtt-dispatcher/dispatcher"
	"go-mqtt-dispatcher/pkg/logger"
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
	debugFlag      = flag.Bool("debug", false, "enable debug mode")
	logLevel       = flag.String("log-level", "info", "log level (debug, info, warn, error)")
)

func main() {
	flag.Parse()

	// Initialize logger
	logger, err := logger.NewLogger(logger.Config{
		Development: *debugFlag,
		Level:       *logLevel,
	})
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	defer logger.Sync()

	// Print version information in a structured way
	logger.Info("===========================================")
	logger.Info(fmt.Sprintf("Starting %s", AppName))
	logger.Info("===========================================")
	logger.Info("Version information",
		"version", Version,
		"commit", Commit,
		"buildTime", BuildTime)
	logger.Info("===========================================")
	logger.Info("For more information, visit: https://github.com/dhcgn/go-mqtt-dispatcher")
	logger.Info("===========================================")

	if *configPathFlag == "" {
		logger.Error("Config file path is required")
		flag.PrintDefaults()
		return
	}

	// Load config
	loggerConfigContext := logger.Named("config")
	config, err := config.LoadConfig(*configPathFlag)
	if err != nil {
		loggerConfigContext.Errorw("Failed to load config", "error", err)
		return
	}

	if *configCheck {
		loggerConfigContext.Info("Config check successful")
		return
	}

	// Create MQTT client
	client, err := connect(AppName+Version+Commit+BuildTime, config.Mqtt.BrokerAsUri)
	if err != nil {
		logger.Named("mqtt-factory").DPanicw("Failed to connect to MQTT broker", "error", err)
		return
	}
	mqttClient := dispatcher.NewPahoMqttClient(client, logger.Named("mqtt-client"))

	d, err := dispatcher.NewDispatcher(&config.DispatcherEntries, mqttClient, logger.Named("dispatcher"))
	if err != nil {
		logger.Named("NewDispatcher").Panicw("Failed to create dispatcher", "error", err)
	}
	d.Run()

	select {}
}

func connect(clientId string, uri *url.URL) (mqtt.Client, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s", uri.Host))
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
