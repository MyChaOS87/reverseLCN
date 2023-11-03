package publisher

import "context"

type Publisher interface {
	Run(ctx context.Context, cancel context.CancelFunc)
	ToTopic(topic string) Topic
}

type Topic interface {
	PublishString(s string)
	Publish(i interface{})
}
