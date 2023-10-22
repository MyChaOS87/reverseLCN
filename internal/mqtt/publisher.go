package mqtt

import (
	"context"
	"encoding/json"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/MyChaOS87/reverseLCN.git/pkg/log"
)

type Publisher interface {
	Run(ctx context.Context, cancel context.CancelFunc)
	ToTopic(topic string) Topic
}

type Topic interface {
	PublishString(s string)
	Publish(i interface{})
}

type topic struct {
	topic  string
	client mqtt.Client
}

type publisher struct {
	client mqtt.Client
}

func (p *publisher) ToTopic(topicName string) Topic {
	return &topic{
		topic:  topicName,
		client: p.client,
	}
}

func (p *publisher) Run(ctx context.Context, cancel context.CancelFunc) {
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

func (t *topic) publishInternal(data string) {
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

func (t *topic) PublishString(s string) {
	t.publishInternal(s)
}

func (t *topic) Publish(data interface{}) {
	b, err := json.Marshal(data)
	if err != nil {
		log.Errorf("Error on Publish, cannot marshal JSON: %s", err)
		return
	}

	t.publishInternal(string(b))
}

func NewPublisher(options ...Option) Publisher {
	config := newDefaultConfig()

	for _, opt := range options {
		opt(config)
	}

	client := mqtt.NewClient(config.clientOptions)

	return &publisher{
		client: client,
	}
}
