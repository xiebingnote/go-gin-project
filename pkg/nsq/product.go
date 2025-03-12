package nsq

import (
	"fmt"
	"math/rand"

	"github.com/xiebingnote/go-gin-project/library/common"
	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/resource"
	pkgproto "github.com/xiebingnote/go-gin-project/pkg/proto"

	"github.com/nsqio/go-nsq"
)

// Producer sends a message to a randomly selected NSQ producer.
//
// It creates a message with predefined properties, serializes it, and publishes
// it to the specified NSQ topic. If the NSQ producer is unavailable or if the
// message fails to publish, the function returns an error.
func Producer() error {
	// Create a message with predefined properties
	message := &pkgproto.TestMessage{
		Id:   1,
		Name: "testName",
	}

	// Serialize the message into a byte slice
	serializedData, err := common.SerializeData(message)
	if err != nil {
		return err
	}

	// Ensure the NSQ producer list is not nil
	if resource.NsqProducer == nil {
		return fmt.Errorf("nsq producer is nil")
	}

	// Select a random NSQ producer from the list
	var producer *nsq.Producer
	if len(resource.NsqProducer) == 1 {
		producer = resource.NsqProducer[0]
	} else {
		producer = resource.NsqProducer[rand.Intn(len(resource.NsqProducer))]
	}

	// Publish the serialized message to the specified NSQ topic
	if err := producer.Publish(config.NsqConfig.NSQ.Consumer.Topic, serializedData); err != nil {
		// Log an error if message publication fails
		resource.LoggerService.Error(fmt.Sprintf("failed to publish message to topic %s, err: %v", config.NsqConfig.NSQ.Consumer.Topic, err))
		return err
	}

	// Stop the producer
	producer.Stop()
	return nil
}
