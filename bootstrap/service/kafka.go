package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"project/library/config"
	"project/library/resource"

	"github.com/IBM/sarama"
)

func InitKafka(_ context.Context) {
	err := InitKafkaClient()
	if err != nil {
		panic(err.Error())
	}
}

func InitKafkaClient() error {
	cfg := config.KafkaConfig

	//---------------- 生产者初始化 ----------------
	producerConfig := sarama.NewConfig()
	producerConfig.Producer.Return.Successes = true
	producerConfig.Producer.Return.Errors = true
	producerConfig.Producer.RequiredAcks = sarama.WaitForAll // 更可靠的确认机制
	producerConfig.Net.DialTimeout = 30 * time.Second        // 增加连接超时

	// 解析 Kafka 版本
	version, err := sarama.ParseKafkaVersion(cfg.Kafka.Version)
	if err != nil {
		return fmt.Errorf("invalid kafka version: %w", err)
	}
	producerConfig.Version = version

	// 创建同步生产者
	resource.KafkaProducer, err = sarama.NewSyncProducer(cfg.Kafka.Brokers, producerConfig)
	if err != nil {
		return fmt.Errorf("kafka producer init failed: %w", err)
	}

	//---------------- 消费者初始化 ----------------
	consumerConfig := sarama.NewConfig()
	consumerConfig.Version = version
	consumerConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	consumerConfig.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{
		sarama.NewBalanceStrategyRange(), // 使用范围分区策略
	}

	// 创建消费者组
	resource.KafkaConsumer, err = sarama.NewConsumerGroup(cfg.Kafka.Brokers, cfg.Kafka.GroupID, consumerConfig)
	if err != nil {
		return fmt.Errorf("kafka consumer init failed: %w", err)
	}

	// 启动后台健康检查
	go kafkaHealthCheck(context.Background())

	log.Printf("Kafka initialized | Brokers: %v | Version: %s", cfg.Kafka.Brokers, cfg.Kafka.Version)
	return nil
}

// 关闭 Kafka 连接
func CloseKafka() error {
	var errs []error

	if resource.KafkaProducer != nil {
		if err := resource.KafkaProducer.Close(); err != nil {
			errs = append(errs, fmt.Errorf("producer close failed: %w", err))
		}
	}

	if resource.KafkaConsumer != nil {
		if err := resource.KafkaConsumer.Close(); err != nil {
			errs = append(errs, fmt.Errorf("consumer close failed: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("kafka shutdown errors: %v", errs)
	}
	return nil
}

// 后台健康检查
func kafkaHealthCheck(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 检查生产者健康状态
			if resource.KafkaProducer != nil {
				if _, _, err := resource.KafkaProducer.SendMessage(&sarama.ProducerMessage{
					Topic: "health_check_topic",
					Value: sarama.StringEncoder("ping"),
				}); err != nil {
					log.Printf("Kafka producer health check failed: %v", err)
				} else {
					log.Println("Kafka producer is healthy")
				}
			}

			// 检查消费者健康状态
			if resource.KafkaConsumer != nil {
				log.Println("Kafka consumer is running")
			}

		case <-ctx.Done():
			return
		}
	}
}

type KafkaConsumerHandler struct {
	Ready chan bool
}

func (h *KafkaConsumerHandler) Setup(sarama.ConsumerGroupSession) error {
	close(h.Ready)
	return nil
}

func (h *KafkaConsumerHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *KafkaConsumerHandler) ConsumeClaim(
	session sarama.ConsumerGroupSession,
	claim sarama.ConsumerGroupClaim,
) error {
	for message := range claim.Messages() {
		log.Printf("Message received: Topic=%s Partition=%d Offset=%d Key=%s Value=%s",
			message.Topic, message.Partition, message.Offset,
			string(message.Key), string(message.Value))

		// 标记消息已处理
		session.MarkMessage(message, "")
	}
	return nil
}

// 启动消费者
func StartKafkaConsumer(topics []string) {
	handler := &KafkaConsumerHandler{
		Ready: make(chan bool),
	}

	go func() {
		for {
			if err := resource.KafkaConsumer.Consume(context.Background(), topics, handler); err != nil {
				log.Printf("Consumer error: %v", err)
			}
		}
	}()

	// 等待消费者准备就绪
	<-handler.Ready
	log.Println("Kafka consumer up and running...")
}
