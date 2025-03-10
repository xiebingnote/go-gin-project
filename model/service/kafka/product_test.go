package kafka

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/xiebingnote/go-gin-project/bootstrap/service"
	"github.com/xiebingnote/go-gin-project/library/config"

	"github.com/BurntSushi/toml"
)

func init() {
	// Retrieve the current working directory
	rootDir, err := os.Getwd()
	if err != nil {
		// Panic if there is an error getting the working directory
		panic(err)
	}

	// Extract the root directory path by splitting on "/model"
	dir := strings.Split(rootDir, "/model")
	rootDir = dir[0]

	// Load MySQL configuration from the specified TOML file
	if _, err := toml.DecodeFile(rootDir+"/conf/service/kafka.toml", &config.KafkaConfig); err != nil {
		// Panic if the MySQL configuration file cannot be decoded
		panic("Failed to load MySQL configuration file: " + err.Error())
	}

	// Initialize the MySQL service with a background context
	service.InitKafka(context.Background())
}

func TestProducer_Success(t *testing.T) {
	err := Producer()
	if err != nil {
		fmt.Println("Failed to produce message:", err)
	} else {
		fmt.Println("Message produced successfully.")
	}
	return
}

//func TestProducer_SerializationError(t *testing.T) {
//	// 设置
//	mockProducer := new(MockKafkaProducer)
//	resource.KafkaProducer = mockProducer
//
//	// 模拟序列化函数返回错误
//	mock.On("SerializeData", mock.Anything).Return(nil, errors.New("serialization error"))
//
//	err := Producer()
//
//	assert.Error(t, err)
//	assert.Equal(t, "serialization error", err.Error())
//}
//
//func TestProducer_SendMessageError(t *testing.T) {
//	// 设置
//	mockProducer := new(MockKafkaProducer)
//	resource.KafkaProducer = mockProducer
//	mockProducer.On("SendMessage", mock.Anything).Return(0, int64(0), errors.New("send message error"))
//	mockProducer.On("Close").Return(nil)
//
//	// 模拟序列化函数
//	mock.On("SerializeData", mock.Anything).Return([]byte{1, 2, 3}, nil)
//
//	err := Producer()
//
//	assert.Error(t, err)
//	assert.Equal(t, "send message error", err.Error())
//	mockProducer.AssertExpectations(t)
//	mock.AssertExpectationsForObjects(t, mockProducer)
//}
