package mqttrouter

import (
	"testing"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/lunny/log"
)

var p *MQTTProcessor

func init() {
	opt := mqtt.NewClientOptions()
	if processor, err := NewProcessor(opt); err != nil {
		panic(err)
	} else {
		p = processor
	}
	p.AddRoute("camera/c3r-125/:sn/:group", 0, MQTTHandler)
}

func MQTTHandler(ps Params, c mqtt.Client, m mqtt.Message) {
	log.Infof("sn: %s, group: %s", ps.ByName("sn"), ps.ByName("group"))
}

func BenchmarkProcess(b *testing.B) {
	for i := 0; i < b.N; i++ {
		p.process("hello/dg/gp", nil, nil)
	}
}

func TestProcess(t *testing.T) {
	p.process("camera/c3r-125/S_12345678/g01", nil, nil)
}
