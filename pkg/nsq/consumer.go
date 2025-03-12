package nsq

import (
	"fmt"

	"github.com/xiebingnote/go-gin-project/library/common"
	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/resource"
	pkgproto "github.com/xiebingnote/go-gin-project/pkg/proto"

	"github.com/nsqio/go-nsq"
)

// Consumer initializes and runs the NSQ consumer.
//
// It adds a message handler to the NSQ consumer and then connects the consumer
// to the specified NSQLookupd addresses in the configuration. If any error
// occurs during the connection process, it logs the error and returns it.
//
// Returns:
//   - An error if connecting to NSQLookupd addresses fails.
func Consumer() error {
	// Add a message handler to the NSQ consumer.
	resource.NsqConsumer.AddHandler(nsq.HandlerFunc(MessageHandler))

	// Connect the consumer to the NSQLookupd addresses.
	if err := resource.NsqConsumer.ConnectToNSQLookupds(config.NsqConfig.NSQ.LookupdAddress); err != nil {
		// Log an error if connection to NSQLookupd fails.
		resource.LoggerService.Error(fmt.Sprintf("failed to connect to nsq lookupds, err: %v", err))
		return err
	}

	return nil
}

// MessageHandler processes a message received from NSQ.
//
// It deserializes the message body into a TestMessage structure and performs
// additional processing logic. If deserialization fails, it returns an error.
//
// Parameters:
//   - message: The NSQ message to process.
//
// Returns:
//   - An error if deserialization fails.
func MessageHandler(message *nsq.Message) error {
	// Deserialize the message body into a TestMessage structure
	data, err := common.DeSerializeData(message.Body, &pkgproto.TestMessage{})
	if err != nil {
		// Return an error if deserialization fails
		return err
	}

	// Additional processing logic
	fmt.Println("data:", data)

	return nil
}
