package kafka

import (
	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/resource"
	pkgproto "github.com/xiebingnote/go-gin-project/pkg/proto"

	"github.com/IBM/sarama"
	"google.golang.org/protobuf/proto"
)

type TestMessage struct {
	Id   int32
	Name string
}

func SerializeData(data *pkgproto.TestMessage) ([]byte, error) {
	return proto.Marshal(data)
}

func Producer() error {
	// 创建一个消息
	message := &pkgproto.TestMessage{
		Id:   1,
		Name: "testName",
	}

	// 序列化消息
	serializedData, err := SerializeData(message)
	if err != nil {
		return err
	}

	msg := &sarama.ProducerMessage{
		Topic: config.KafkaConfig.Kafka.ProducerTopic,
		Value: sarama.ByteEncoder(serializedData),
	}

	if _, _, err := resource.KafkaProducer.SendMessage(msg); err != nil {
		return err
	}
	defer resource.KafkaProducer.Close()

	return nil
}
