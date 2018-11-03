package mqtt

import (
	"path"
	"time"

	"github.com/JohannWeging/logerr"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/juju/errors"
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

func New(opts *Options) *MQTT {
	m := &MQTT{
		opts: opts,
		logger: log.WithFields(log.Fields{
			"module":      "mqtt",
			"mqtt_broker": opts.Broker,
		}),
		Events: make(chan *Event),
	}

	co := mqtt.NewClientOptions()
	co.AddBroker(opts.Broker)
	co.SetUsername(opts.User)
	co.SetPassword(opts.Password)
	co.SetOnConnectHandler(m.onConnect)

	m.client = mqtt.NewClient(co)
	return m
}

func (m *MQTT) onConnect(client mqtt.Client) {
	m.logger.Info("connected to mqtt broker")

	cmdTopic := path.Join("cmnd", m.opts.Topic, "#")
	logger := m.logger.WithField("cmd_topic", cmdTopic)
	token := client.Subscribe(cmdTopic, byte(m.opts.QoS), m.cmndHandler)
	logger.Debug("subcribe to cmd topic")

	if !token.WaitTimeout(10 * time.Second) {
		err := errors.New("token timeout")
		logger.WithError(err).Error("failed to subscribe to topic: reached timeout")
	}

	err := token.Error()
	if err != nil {
		logger.WithError(err).Error("failed to subscribe to topic: token error")
	}
}

func (m *MQTT) cmndHandler(client mqtt.Client, msg mqtt.Message) {
	if msg.Duplicate() {
		m.logger.Debug("recived duplicated message")
		return
	}

	m.Events <- &Event{
		Topic:   msg.Topic(),
		Payload: string(msg.Payload()),
	}
}

func (m *MQTT) Connect() (err error) {
	logFields := log.Fields{"mqtt_broker": m.opts.Broker}
	logerr.DeferWithFields(&err, logFields)
	errors.DeferredAnnotatef(&err, "failed to connect to broker")

	log.WithFields(logFields).Debug("connect to broker")
	token := m.client.Connect()
	ok := token.WaitTimeout(10 * time.Second)
	if !ok {
		return errors.New("connect timeout")
	}
	return token.Error()
}

func (m *MQTT) Disconnect() {
	m.client.Disconnect(300)
}

func (m *MQTT) Publish(endpoint, payload string) (err error) {
	topic := path.Join("stat", m.opts.Topic, endpoint)
	logFields := log.Fields{"mqtt_broker": m.opts.Broker, "mqtt_topic": topic, "mqtt_payload": payload}
	logerr.DeferWithFields(&err, logFields)
	errors.DeferredAnnotatef(&err, "failed to publish mqtt msg")

	if !m.client.IsConnected() {
		return errors.New("not connected")
	}

	m.logger.WithFields(logFields).Debug("publish mqtt msg")

	token := m.client.Publish(topic, byte(m.opts.QoS), m.opts.Retain, payload)

	ok := token.WaitTimeout(10 * time.Second)
	if !ok {
		return errors.New("publish timeout")
	}

	return token.Error()
}
