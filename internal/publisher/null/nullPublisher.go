package null

import (
	"context"

	"github.com/MyChaOS87/reverseLCN.git/internal/publisher"
)

type (
	nullPublisher struct{}
	nullTopic     struct{}
)

var (
	_ publisher.Publisher = &nullPublisher{}
	_ publisher.Topic     = &nullTopic{}
)

func (nullPublisher) Run(context.Context, context.CancelFunc) {
}

func (nullPublisher) ToTopic(string) publisher.Topic {
	return &nullTopic{}
}

func (nullTopic) Publish(interface{}) {
}

func (nullTopic) PublishString(string) {}

func NewPublisher() publisher.Publisher {
	return &nullPublisher{}
}
