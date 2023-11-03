package null

import (
	"context"

	"github.com/MyChaOS87/reverseLCN.git/internal/broker"
)

type (
	nullPublisher struct{}
	nullTopic     struct{}
)

var (
	_ broker.Publisher = &nullPublisher{}
	_ broker.Topic     = &nullTopic{}
)

func (nullPublisher) Run(context.Context, context.CancelFunc) {
}

func (nullPublisher) Topic(string) broker.Topic {
	return &nullTopic{}
}

func (nullTopic) Publish(interface{}) {
}

func (nullTopic) PublishString(string) {}

func (nullTopic) Subscribe(hint interface{}, callback broker.CallbackFunction) {
}

func NewPublisher() broker.Publisher {
	return &nullPublisher{}
}
