package broker

import "context"

type Publisher interface {
	Run(ctx context.Context, cancel context.CancelFunc)
	Topic(topic string) Topic
}

type Topic interface {
	PublishString(s string)
	Publish(i interface{})
	Subscribe(hint interface{}, callback CallbackFunction)
}

type CallbackFunction func(data interface{})
