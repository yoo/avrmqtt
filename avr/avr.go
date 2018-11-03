package avr

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/JohannWeging/logerr"
	"github.com/juju/errors"
	log "github.com/sirupsen/logrus"
	"github.com/ziutek/telnet"
)

const (
	urlAppDirekt = "/goform/formiPhoneAppDirect.xml"
)

type Event interface {
	Type() string
}

type ConnectEvent struct {
	State string
}

func (c *ConnectEvent) Type() string {
	return "connect"
}

type TelnetEvent struct {
	Data string
}

func (t *TelnetEvent) Type() string {
	return "telnet"
}

type AVR struct {
	m      sync.Mutex
	opts   *Options
	http   *http.Client
	telnet *telnet.Conn
	Events chan Event
}

type Options struct {
	Host         string
	HttpPort     string
	TelnetPort   string
	httpEndpoint string
}

func New(opts *Options) *AVR {
	opts.httpEndpoint = fmt.Sprintf("http://%s:%s%s", opts.Host, opts.HttpPort, urlAppDirekt)
	avr := &AVR{
		opts:   opts,
		http:   http.DefaultClient,
		Events: make(chan Event),
	}

	go avr.listenTelnet()
	return avr
}

func (a *AVR) listenTelnet() {
	telnetHost := fmt.Sprintf("%s:%s", a.opts.Host, a.opts.TelnetPort)
	fields := log.Fields{
		"telnet_host": telnetHost,
		"module":      "telnet",
	}
	logger := log.WithFields(fields)
	var err error
	for {
		a.telnet, err = telnet.DialTimeout("tcp", telnetHost, 5*time.Second)
		if err != nil {
			// this is set to info because if the receiver is powered down
			// is can spam logs
			logger.WithError(err).Info("failed to connect to telnet")
			continue
		}
		logger.Debug("telnet connected")
		a.Events <- &ConnectEvent{State: "connect"}
		for {
			data, err := a.telnet.ReadString('\r')
			if err != nil {
				logger.Errorf("failed to read form telnet")
				break
			}
			data = strings.Trim(data, " \n\r")
			logger.WithField("data", data).Debug("recived data")
			a.Events <- &TelnetEvent{Data: data}
		}
		a.Events <- &ConnectEvent{State: "disconnect"}
	}
}

func (a *AVR) Command(cmd string) error {
	a.m.Lock()
	defer a.m.Unlock()
	err := get(a.http, a.opts.httpEndpoint, cmd)
	err = logerr.WithFields(err,
		logerr.Fields{
			"cmd":    cmd,
			"module": "telnet",
		},
	)

	time.Sleep(100 * time.Millisecond)
	return errors.Annotate(err, "failed to send cmd")
}

func get(client *http.Client, endpoint string, cmd string) error {
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		err = logerr.WithField(err, "url", endpoint)
		err = errors.Annotate(err, "failed to create request")
		return err
	}

	// add the command as empty parameter
	req.URL.RawQuery = url.QueryEscape(cmd)
	log.WithFields(log.Fields{
		"module": "telnet",
		"url":    req.URL.String(),
	}).Debug("send http request")
	_, err = client.Do(req)
	err = logerr.WithField(err, "url", req.URL.String())
	return errors.Annotate(err, "failed to do request")
}
