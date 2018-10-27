[![Build Status](https://ci.sysexit.com/api/badges/JohannWeging/avrmqtt/status.svg)](https://ci.sysexit.com/JohannWeging/avrmqtt) [![Docker Hub](https://img.shields.io/badge/docker-container-blue.svg?longCache=true&style=flat-square)](https://hub.docker.com/r/johannweging/avrmqtt) [![GitHub](https://img.shields.io/badge/github-repo-blue.svg?longCache=true&style=flat-square)](https://github.com/JohannWeging/avrmqtt)

# avrmqtt
avrmqtt Connects a Denon AVR Receiver to a MQTT Brooker.
The MQTT implementation supports most of the telnet commands specified
in the [Denon AVR control protocol](https://www.denon.de/de/product/hometheater/avreceivers/avrx1000?docname=AVRX1000_E300_PROTOCOL(1000)_V04.pdf).

## Usage
Example:
```
avrmqtt --avr-host avr.lan --mqtt-host broker.lan --mqtt-user fancypants --mqtt-password foobar
```

Or with Docker:
```
docker run -e AVR_HOST=avr.lan \
  -e MQTT_HOST=broker.lan \
  -e MQTT_USER=fancypants \
  -e MQTT_PASSWORD=foobar \
  johannweging/avrmqtt
```

Full usage:
```
avrmqtt --help
Usage of avrmqtt:
      --avr-host string          denon avr host name (default "denon")
      --avr-http-port string     denon avr telnet port (default "80")
      --avr-telnet-port string   denon avr telnet port (default "23")
      --log-level string         log level (fatal|error|warn|info|debug)
      --mqtt-host string         mqtt host name
      --mqtt-password string     mqtt user password
      --mqtt-topic string        mqtt topic (default "denon")
      --mqtt-user string         mqtt user name
pflag: help requested

```

## MQTT Topics
In general the MQTT topics use the function as the topic and the parameter as payload.
Commands are send via the `cmnd` topic. The receiver will answer to changes via the `stat` topic.

With the default mqtt topic `avr`.
```
# setting the master volume to 30
# topic     payload
cmnd/avr/MV 300

# the receiver response with
stat/avr/MV 300
```

### MQTT Topic Exceptions
Two functions are handled differently.

#### CV Function
The CV functions is spilt into sub commands at the first whitespace.
```
# topic            payload
cmnd/${topic}/CVFL UP
stat/${topic}/CVFL UP
[...]
cmnd/${topic}/CVC DOWN
stat/${topic}/CVC DOWN
[...]
```

#### PS Function
The PS function is either split at the first whitespace or `:`.
```
# topic                payload
cmnd/${topic}/PSMULTEQ AUDYSSEY
[...]
cmnd/${topic}/PSDYNEQ ON
[...]
```

### Implementation, Limitations and Wired Behaviour
The `cmnd` topic uses the http interface of the receiver:
`http://${host}/goform/formiPhoneAppDirect.xml"`
The MQTT commands are translated to:
```
# topic          payload
cmnd/${topic}/MV 300
http://${host}/goform/formiPhoneAppDirect.xml&MV300

cmnd/${topic}/PSDYNEQ ON
http://${host}/goform/formiPhoneAppDirect.xml&PSDYNEQ+ON

cmnd/${topic}/SI SAT/CBL
http://${host}/goform/formiPhoneAppDirect.xml&SISAT%2FCBL
```
Only commands supported by the http interface work with avrmqtt.

The `stat` topic  is set by the data send over telnet.
```
# topic          payload
MV300<CR>
stat/${topic}/MV 300

PSDYNEQ ON<CR>
stat/${topic}/PSDYNEQ ON

SISAT/CBL<CR>
stat/${topic}/SI SAT/CBL
```

Some commands are synonyms of each other, which can lead to
diffrent results on the `stat` topic.
```
# topic          payload
cmnd/${topic}/MS MOVIE
# results in
stat/${topic}/MS DOLBY PL2 C
```

## Automation Example with Home Assistant
The drop down menu `tv_profile` triggers different profiles for the receiver.
Here it loads the `Music` profile.
```yaml
- alias: set_tv_music
  trigger:
    platform: state
    entity_id: input_select.tv_profile
  condition:
    condition: state
    entity_id: input_select.tv_profile
    state: "Music"
  action:
    # Play stereo on all speakers
    - service: mqtt.publish
      data:
        topic: cmnd/avr/MS
        payload: MCH STEREO
    # Disable dynamic volume compression
    - service: mqtt
      data:
        topic: cmnd/avr/PSDYNVOL
        payload: "OFF"
    # Set the sound restorer to medium
    - service: mqtt.publish
      data:
        topic: cmnd/avr/PSRSTR
        payload: MED
    # Enable the dynamic equalizer
    - service: mqtt.publish
      data:
        topic: cmnd/avr/PSDYNEQ
        payload: "ON"
```

