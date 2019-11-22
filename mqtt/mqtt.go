package mqtt

import (
	"fmt"
	"path"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
)

type Event struct {
	Topic   string
	Payload string
}

type MQTT struct {
	client mqtt.Client
	opts   *Options
	logger *log.Entry
	Events chan *Event
}

type Options struct {
	Broker   string
	User     string
	Password string
	Topic    string
	QoS      int
	Retain   bool
}

func New(opts *Options) (*MQTT, error) {
	m := &MQTT{
		opts: opts,
		logger: log.WithFields(log.Fields{
			"module":      "mqtt",
			"mqtt_broker": opts.Broker,
		}),
		Events: make(chan *Event),
	}
	m.logger = log.WithFields(m.logFields())

	co := mqtt.NewClientOptions()
	co.AddBroker(opts.Broker)
	co.SetUsername(opts.User)
	co.SetPassword(opts.Password)
	co.SetOnConnectHandler(m.onConnect)
	co.SetAutoReconnect(true)

	m.client = mqtt.NewClient(co)
	token := m.client.Connect()
	ok := token.WaitTimeout(10 * time.Second)
	if !ok {
		return nil, fmt.Errorf("connect timeout %q", opts.Broker)

	}
	if err := token.Error(); err != nil {
		return nil, fmt.Errorf("failed to connect to %q: %w", opts.Broker, err)
	}
	return m, nil
}

func (m *MQTT) logFields() map[string]interface{} {
	return map[string]interface{}{
		"module": "mqtt",
		"broker": m.opts.Broker,
	}
}

func (m *MQTT) onConnect(client mqtt.Client) {
	m.logger.Info("connected to mqtt broker")

	cmdTopic := path.Join("cmnd", m.opts.Topic, "#")
	logger := m.logger.WithField("cmd_topic", cmdTopic)
	token := client.Subscribe(cmdTopic, byte(m.opts.QoS), m.cmndHandler)
	logger.Debug("subcribe to cmd topic")

	if !token.WaitTimeout(10 * time.Second) {
		err := fmt.Errorf("token timeout")
		logger.WithError(err).Error("failed to subscribe to topic: reached timeout")
		return
	}

	err := token.Error()
	if err != nil {
		logger.WithError(err).Error("failed to subscribe to topic: token error")
		return
	}
}

func (m *MQTT) cmndHandler(client mqtt.Client, msg mqtt.Message) {
	if msg.Duplicate() {
		m.logger.Debug("recived duplicated message")
		return
	}
	m.logger.WithFields(log.Fields{
		"topic":   msg.Topic(),
		"payload": string(msg.Payload()),
	}).Debug("recived cmnd msg %q from %q", string(msg.Payload()), msg.Topic())

	m.Events <- &Event{
		Topic:   msg.Topic(),
		Payload: string(msg.Payload()),
	}
}

func (m *MQTT) Publish(endpoint, payload string) (err error) {
	topic := path.Join("stat", m.opts.Topic, endpoint)
	defer func() {
		if err == nil {
			return
		}
		err = fmt.Errorf("failed to publish mqtt msg %q to %q: %s", payload, payload, err)
	}()

	if !m.client.IsConnected() {
		return fmt.Errorf("not connected")
	}

	m.logger.Debug("publish mqtt msg %q to %q", payload, endpoint)

	token := m.client.Publish(topic, byte(m.opts.QoS), m.opts.Retain, payload)

	ok := token.WaitTimeout(10 * time.Second)
	if !ok {
		return fmt.Errorf("publish timeout")
	}

	return token.Error()
}
