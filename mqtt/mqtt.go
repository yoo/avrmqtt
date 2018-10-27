package mqtt

import (
	"path"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/juju/errors"
	log "github.com/sirupsen/logrus"

	"github.com/JohannWeging/avrmqtt/avr"
)

type MQTT struct {
	Client     mqtt.Client
	opts       *Options
	logger     *log.Entry
	disconnect chan struct{}
}

type Options struct {
	Broker   string
	User     string
	Password string
	Topic    string
	QoS      int
	Retain   bool

	AVR *avr.AVR
}

func New(opts *Options) (*MQTT, error) {
	m := &MQTT{
		opts: opts,
		logger: log.WithFields(log.Fields{
			"module":      "mqtt",
			"mqtt_broker": opts.Broker,
		}),
	}

	co := mqtt.NewClientOptions()
	co.AddBroker(opts.Broker)
	co.SetUsername(opts.User)
	co.SetPassword(opts.Password)
	co.SetOnConnectHandler(m.onConnect)
	co.SetConnectionLostHandler(m.connectionLost)

	m.Client = mqtt.NewClient(co)
	token := m.Client.Connect()
	token.Wait()
	err := token.Error()
	return m, errors.Annotate(err, "failed to connect to broker")
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
	go m.publish()
}

func (m *MQTT) connectionLost(clien mqtt.Client, err error) {
	m.logger.Info("disconnect from broker")
	m.disconnect <- struct{}{}
}

func (m *MQTT) cmndHandler(client mqtt.Client, msg mqtt.Message) {
	if msg.Duplicate() {
		m.logger.Debug("recived duplicated message")
		return
	}
	_, endpoint := path.Split(msg.Topic())
	payload := string(msg.Payload())

	cmd := ""
	if strings.HasPrefix(endpoint, "PS") || strings.HasPrefix(endpoint, "CV") {
		if endpoint == "PSMODE" {
			cmd = endpoint + ":" + payload
		} else {
			cmd = endpoint + " " + payload
		}
	} else {
		cmd = endpoint + payload
	}

	m.logger.WithField("command", cmd).Debug("send command")
	err := m.opts.AVR.Command(cmd)
	if err != nil {
		m.logger.WithError(err).Error("failed to forward mqtt command")
	}
}

func logToken(token mqtt.Token) {
	ok := token.WaitTimeout(1 * time.Minute)
	if !ok {
		return
	}
	err := token.Error()
	if err != nil {
		log.WithError(err).Error("failed to publish message")
	}
}

func (m *MQTT) publish() {

	for {
		select {
		case e := <-m.opts.AVR.Events:
			topic := path.Join("stat", m.opts.Topic, e.Type)
			m.logger.WithFields(log.Fields{
				"mqtt_topic":   topic,
				"mqtt_payload": e.Value,
			}).Debug("publish avr event")
			token := m.Client.Publish(topic, byte(m.opts.QoS), m.opts.Retain, e.Value)
			go logToken(token)
		case <-m.disconnect:
			return
		}
	}
}
