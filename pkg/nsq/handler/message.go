package handler

import (
	"fmt"

	"github.com/xiebingnote/go-gin-project/library/common"
	pkgproto "github.com/xiebingnote/go-gin-project/pkg/proto"

	"github.com/nsqio/go-nsq"
)

// HandleMessage deserializes the application message and performs its work.
func HandleMessage(message *nsq.Message) error {
	data, err := common.DeSerializeData(message.Body, &pkgproto.TestMessage{})
	if err != nil {
		return err
	}
	fmt.Println("data:", data)
	return nil
}
