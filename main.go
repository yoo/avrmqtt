package main

import (
	"github.com/luzifer/rconfig"
	log "github.com/sirupsen/logrus"

	"github.com/JohannWeging/avrmqtt/avr"
	"github.com/JohannWeging/avrmqtt/mqtt"
)

var conf = struct {
	AVRHost       string `flag:"avr-host" default:"avr" description:"denon avr host name"`
	AVRHTTPPort   string `flag:"avr-http-port" env:"AVR_HTTP_PORT" default:"80" description:"denon avr telnet port"`
	AVRTelnetPort string `flag:"avr-telnet-port" default:"23" description:"denon avr telnet port"`

	MQTTBroker   string `flag:"mqtt-broker" description:"mqtt host name"`
	MQTTUser     string `flag:"mqtt-user" description:"mqtt user name"`
	MQTTPassword string `flag:"mqtt-password" description:"mqtt user password"`
	MQTTTopic    string `flag:"mqtt-topic" default:"avr" description:"mqtt topic"`
	MQTTRetain   bool   `flag:"mqtt-retain" default:"false" description:"set retain for messages to stat/$topic/#"`

	LogLevel string `flag:"log-level" default:"info" description:"log level (fatal|error|warn|info|debug)"`
}{}

func main() {

	rconfig.AutoEnv(true)
	if err := rconfig.Parse(&conf); err != nil {
		log.WithError(err).Fatal("failed to parse config")

	}

	lvl, err := log.ParseLevel(conf.LogLevel)
	if err != nil {
		log.WithError(err).Fatal("failed to parse log level")
	}
	log.SetLevel(lvl)

	avrOpts := &avr.Options{
		Host:       conf.AVRHost,
		HttpPort:   conf.AVRHTTPPort,
		TelnetPort: conf.AVRTelnetPort,
	}

	reciver := avr.New(avrOpts)

	mqttOpts := &mqtt.Options{
		Broker:   conf.MQTTBroker,
		User:     conf.MQTTUser,
		Password: conf.MQTTPassword,
		QoS:      1,
		Topic:    conf.MQTTTopic,
		Retain:   conf.MQTTRetain,
		AVR:      reciver,
	}
	_, err = mqtt.New(mqttOpts)
	if err != nil {
		log.WithError(err).Fatal("failed to connect to mqtt broker")
	}

	// just wait
	select {}
}
