package main

import (
	"path"
	"sync"

	"github.com/luzifer/rconfig"
	log "github.com/sirupsen/logrus"

	"github.com/yoo/avrmqtt/avr"
	"github.com/yoo/avrmqtt/mqtt"
)

var logLock sync.Mutex
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

	receiver := avr.New(avrOpts)

	mqttOpts := &mqtt.Options{
		Broker:   conf.MQTTBroker,
		User:     conf.MQTTUser,
		Password: conf.MQTTPassword,
		QoS:      1,
		Topic:    conf.MQTTTopic,
		Retain:   conf.MQTTRetain,
	}
	broker, err := mqtt.New(mqttOpts)
	if err != nil {
		log.WithError(err).Fatal("failed to connect to mqtt broker")
	}
	run(receiver, broker)
}

func run(receiver *avr.AVR, broker *mqtt.MQTT) {
	for {
		select {
		case e := <-receiver.Events:
			go telnetEvent(e, broker)
		case e := <-broker.Events:
			go brokerEvent(e, receiver)
		}
	}
}

func brokerEvent(event *mqtt.Event, receiver *avr.AVR) {
	_, endpoint := path.Split(event.Topic)
	err := receiver.Command(endpoint, event.Payload)
	errLog(err, "failed to send mqtt event to receiver")
}

func telnetEvent(event *avr.Event, broker *mqtt.MQTT) {
	endpoint, payload := parseData(event.Data)
	err := broker.Publish(endpoint, payload)
	errLog(err, "failed to publish telnet event to mqtt")
}

func errLog(err error, msg string) {
	if err == nil {
		return
	}
	logLock.Lock()
	defer logLock.Unlock()
	log.WithError(err).Error(msg)
}
