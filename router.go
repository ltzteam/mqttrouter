package mqttrouter

import (
	"fmt"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/lunny/log"
)

type MQTTProcessor struct {
	tree   *node
	topics []string
	cli    mqtt.Client
	mu     sync.Mutex
}

type Param struct {
	Key   string
	Value string
}

type Params []Param

func (ps Params) ByName(name string) string {
	for i := range ps {
		if ps[i].Key == name {
			return ps[i].Value
		}
	}
	return ""
}

type Handle func(Params, mqtt.Client, mqtt.Message)

func NewProcessor(opt *mqtt.ClientOptions) (*MQTTProcessor, error) {
	processor := &MQTTProcessor{
		tree: &node{},
		cli:  mqtt.NewClient(opt),
	}

	token := processor.cli.Connect()
	if token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}
	log.Info("mqtt connected.")

	go func() {
		i := 0
		for {
			log.Infof("mqtt is connect: %v", processor.cli.IsConnected())
			time.Sleep(1 * time.Second)
			i++
			log.Infof("error: %v", processor.Publish("camera/c3r-125/S_12345678/check", 2, false, fmt.Sprintf("%d", i)))
		}
	}()

	return processor, nil
}

func (r *MQTTProcessor) process(topic string, cli mqtt.Client, message mqtt.Message) {
	if handle, ps, _ := r.tree.getValue(topic); handle != nil {
		handle(ps, cli, message)
	}
}

func toTopic(url string) string {
	for _, item := range strings.Split(url, "/") {
		if strings.HasPrefix(item, ":") {
			url = strings.Replace(url, item, "+", 1)
		}
	}
	return url
}

func (r *MQTTProcessor) AddRoute(url string, qos byte, handle Handle) {
	topic := toTopic(url)
	log.Infof("topic: %s", topic)

	r.mu.Lock()
	r.tree.addRoute(url, handle)
	r.topics = append(r.topics, topic)
	r.mu.Unlock()

	r.cli.Subscribe(topic, qos, func(c mqtt.Client, m mqtt.Message) {
		r.process(m.Topic(), c, m)
	})
}

func (r *MQTTProcessor) Unsubscribe(url string) error {
	topic := toTopic(url)
	if token := r.cli.Unsubscribe(topic); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	r.mu.Lock()
	for i, t := range r.topics {
		if strings.Compare(t, topic) == 0 {
			r.topics[i] = r.topics[len(r.topics)-1]
			r.topics[len(r.topics)-1] = ""
			r.topics = r.topics[:len(r.topics)-1]
		}
	}
	r.mu.Unlock()
	return nil
}

func (r *MQTTProcessor) Close() {
	if r.cli.IsConnected() {
		for _, t := range r.topics {
			r.Unsubscribe(t)
		}
		r.cli.Disconnect(3000)
	}
}

func (r *MQTTProcessor) Publish(topic string, qos byte, retained bool, payload interface{}) error {
	if token := r.cli.Publish(topic, qos, retained, payload); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}
