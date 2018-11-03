package main

import (
	"path"
	"strings"
	"sync"

	"github.com/JohannWeging/logerr"
	"github.com/luzifer/rconfig"
	log "github.com/sirupsen/logrus"

	"github.com/JohannWeging/avrmqtt/avr"
	"github.com/JohannWeging/avrmqtt/mqtt"
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
	broker := mqtt.New(mqttOpts)
	if err != nil {
		f := logerr.GetFields(err)
		log.WithFields(f).WithError(err).Fatal("failed to connect to mqtt broker")
	}
	run(receiver, broker)
}

func run(receiver *avr.AVR, broker *mqtt.MQTT) {
	for {
		select {
		case e := <-receiver.Events:
			go receiverEvent(e, broker)
		case e := <-broker.Events:
			go brokerEvent(e, receiver)
		}
	}
}

func receiverEvent(event avr.Event, broker *mqtt.MQTT) {
	var err error
	switch e := event.(type) {
	case *avr.ConnectEvent:
		if e.State == "connect" {
			err = broker.Connect()
		} else {
			broker.Disconnect()
		}
	case *avr.TelnetEvent:
		err = telnetEvent(e, broker)
	default:
		panic("unreachable")
	}
	errLog(err, "failed to publish telnet event to mqtt")
}

func brokerEvent(event *mqtt.Event, receiver *avr.AVR) {
	_, endpoint := path.Split(event.Topic)
	cmd := ""
	if strings.HasPrefix(endpoint, "PS") || strings.HasPrefix(endpoint, "CV") {
		if endpoint == "PSMODE" {
			cmd = endpoint + ":" + event.Payload
		} else {
			cmd = endpoint + " " + event.Payload
		}
	} else {
		cmd = endpoint + event.Payload
	}
	err := receiver.Command(cmd)
	errLog(err, "failed to send mqtt event to receiver")
}

func telnetEvent(event *avr.TelnetEvent, broker *mqtt.MQTT) error {
	endpoint, payload := parseData(event.Data)
	return broker.Publish(endpoint, payload)
}

func errLog(err error, msg string) {
	if err == nil {
		return
	}
	logLock.Lock()
	defer logLock.Unlock()
	fields := logerr.GetFields(err)
	log.WithFields(fields).WithError(err).Error(msg)
}
