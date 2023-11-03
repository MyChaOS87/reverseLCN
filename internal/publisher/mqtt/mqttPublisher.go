package mqtt

import (
	"context"
	"encoding/json"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/MyChaOS87/reverseLCN.git/internal/publisher"
	"github.com/MyChaOS87/reverseLCN.git/pkg/log"
)

var (
	_ publisher.Publisher = &mqttPublisher{}
	_ publisher.Topic     = &mqttTopic{}
)

type mqttTopic struct {
	topic  string
	client mqtt.Client
}

type mqttPublisher struct {
	client mqtt.Client
}

func (p *mqttPublisher) ToTopic(topicName string) publisher.Topic {
	return &mqttTopic{
		topic:  topicName,
		client: p.client,
	}
}

func (p *mqttPublisher) Run(ctx context.Context, cancel context.CancelFunc) {
	token := p.client.Connect()
	go func() {
		select {
		case <-token.Done():
			if err := token.Error(); err != nil {
				log.Errorf("Error Connecting to MQTT Broker: %s", err.Error())
				cancel()
			} else {
				log.Infof("Connection to MQTT Broker established")
			}
		case <-ctx.Done():
			p.client.Disconnect(0)
		}
	}()
}

func (t *mqttTopic) publishInternal(data string) {
	token := t.client.Publish(t.topic, 1, false, data)
	go func() {
		token.Wait()
		if err := token.Error(); err != nil {
			log.Errorf("Error on Publish to %s: %s", t.topic, err)
		} else {
			log.Infof("Successfully Published %s to %s", data, t.topic)
		}
	}()
}

func (t *mqttTopic) PublishString(s string) {
	t.publishInternal(s)
}

func (t *mqttTopic) Publish(data interface{}) {
	b, err := json.Marshal(data)
	if err != nil {
		log.Errorf("Error on Publish, cannot marshal JSON: %s", err)
		return
	}

	t.publishInternal(string(b))
}

func NewPublisher(options ...Option) publisher.Publisher {
	config := newDefaultConfig()

	for _, opt := range options {
		opt(config)
	}

	client := mqtt.NewClient(config.clientOptions)

	return &mqttPublisher{
		client: client,
	}
}
