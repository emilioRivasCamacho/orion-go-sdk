package kafka

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/gig/orion-go-sdk/env"
	skafka "github.com/segmentio/kafka-go"
)

// Transport object
type Transport struct {
	listening      bool
	options        Options
	handlers       handlers
	rawMsgHandlers rawMsgHandlers
	close          chan struct{}
	closeHandler   func(error)
}

// Option type
type Option func(Options)

// Options for kafka transport
type Options struct {
	URL                    string
	TopicPartition         int
	TopicReplicationFactor int
	ProducerPartition      int32
	ConsumerGroupID        string
	SocketTimeout          string
	Offset                 string
	Consumer               *kafka.Consumer
	Producer               *kafka.Producer
}

type handlers map[string]func([]byte)
type rawMsgHandlers map[string]func(interface{})

// New returns client for Kafka messaging
func New(options ...Option) *Transport {
	t := new(Transport)

	t.handlers = make(handlers)
	t.rawMsgHandlers = make(rawMsgHandlers)
	t.options.URL = env.Get("KAFKA_HOST", "localhost:9092")
	t.options.ConsumerGroupID = env.Get("KAFKA_GROUP_ID", "default-go")
	t.options.SocketTimeout = env.Get("KAFKA_SOCKET_TIMEOUT_MS", "1000")
	t.options.Offset = env.Get("KAFKA_OFFSET", "latest")
	producerPartition := env.Get("KAFKA_PRODUCER_PARTITION", "-1")
	topicPartition := env.Get("KAFKA_TOPIC_PARTITION", "5")
	topicReplicationFactor := env.Get("KAFKA_TOPIC_REPLICATION_FACTOR", "1")

	i, err := strconv.ParseInt(producerPartition, 10, 32)
	if err != nil {
		panic(err)
	}
	t.options.ProducerPartition = int32(i)

	i, err = strconv.ParseInt(topicPartition, 10, 64)
	if err != nil {
		panic(err)
	}
	t.options.TopicPartition = int(i)

	i, err = strconv.ParseInt(topicReplicationFactor, 10, 64)
	if err != nil {
		panic(err)
	}
	t.options.TopicReplicationFactor = int(i)

	for _, setter := range options {
		setter(t.options)
	}

	if t.options.Producer == nil {
		p, err := kafka.NewProducer(&kafka.ConfigMap{
			"bootstrap.servers": t.options.URL,
			"socket.timeout.ms": t.options.SocketTimeout,
		})
		if err != nil {
			panic(err)
		}
		t.options.Producer = p
	}

	if t.options.Consumer == nil {
		config := &kafka.ConfigMap{
			"bootstrap.servers":  t.options.URL,
			"group.id":           t.options.ConsumerGroupID,
			"auto.offset.reset":  t.options.Offset,
			"enable.auto.commit": false,
		}
		p, err := kafka.NewConsumer(config)
		if err != nil {
			panic(err)
		}
		t.options.Consumer = p
	}

	t.close = make(chan struct{})
	return t
}

// Listen to kafka
func (t *Transport) Listen(callback func()) {
	t.listening = true
	go t.poll(callback)

	sigs := make(chan os.Signal)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		t.close <- struct{}{}
	}()
	<-t.close
	t.listening = false
	t.options.Consumer.Close()
	t.flush()
}

// Publish to topic
func (t *Transport) Publish(topic string, data []byte) error {
	topic = normalizeTopic(topic)
	return t.options.Producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: t.options.ProducerPartition,
		},
		Value: data,
	}, nil)
}

// Subscribe for topic
// Messages will be committed automatically
func (t *Transport) Subscribe(topic, serviceName string, handler func([]byte)) error {
	topic = normalizeTopic(topic)
	t.checkTopic(topic)
	t.handlers[topic] = handler
	return nil
}

// SubscribeForRawMsg for topic
// Messages have te be committed manually
func (t *Transport) SubscribeForRawMsg(topic, serviceName string, handler func(interface{})) error {
	topic = normalizeTopic(topic)
	t.checkTopic(topic)
	t.rawMsgHandlers[topic] = handler
	return nil
}

// Handle path
func (t *Transport) Handle(path string, group string, handler func([]byte, func([]byte))) error {
	println("kafka rpc is not implemented")
	return nil
}

// Request path
func (t *Transport) Request(path string, payload []byte, timeOut int) ([]byte, error) {
	println("kafka rpc is not implemented")
	return nil, nil
}

// Close connection
func (t *Transport) Close() {
	go func() {
		t.close <- struct{}{}
	}()
}

// OnClose adds a handler when the transport is closed. Passes error as an argument
func (t *Transport) OnClose(handler interface{}) {
	if callback, ok := handler.(func(error)); ok {
		t.closeHandler = callback
	}
}

func (t *Transport) flush() {
	t.options.Producer.Flush(1000)
	t.options.Producer.Close()
}

func (t *Transport) poll(callback func()) {
	topics := make([]string, 0, len(t.handlers)+len(t.rawMsgHandlers))
	for k := range t.handlers {
		topics = append(topics, k)
	}
	for k := range t.rawMsgHandlers {
		topics = append(topics, k)
	}

	err := t.createTopics(topics)
	if err != nil {
		panic(err)
	}

	if len(topics) > 0 {
		err = t.options.Consumer.SubscribeTopics(topics, nil)
		if err != nil {
			panic(err)
		}
	}

	go callback()

	for t.listening {
		msg, err := t.options.Consumer.ReadMessage(-1)
		// I had few cases locally where I got panic and it looks like the msg
		// is nil although the error is nil too
		if err == nil && msg != nil {
			topic := *msg.TopicPartition.Topic
			if hanlder, ok := t.handlers[topic]; ok {
				hanlder(msg.Value)
				t.options.Consumer.CommitMessage(msg)
			} else if hanlder, ok := t.rawMsgHandlers[topic]; ok {
				hanlder(msg)
			} else {
				log.Printf("Something went wrong, unable to find handler for topic %s", topic)
			}
		} else {
			if err != nil {
				log.Printf("Error while reading %s", err.Error())
			}
			t.Close()
			break
		}
	}
}

func (t *Transport) createTopics(topics []string) error {
	dialer := &skafka.Dialer{
		Timeout:   10 * time.Second,
		DualStack: true,
	}

	conn, err := dialer.DialContext(context.Background(), "tcp", t.options.URL)
	if err != nil {
		return err
	}

	defer conn.Close()

	configs := make([]skafka.TopicConfig, 0, len(topics))
	for _, topic := range topics {
		configs = append(configs, skafka.TopicConfig{
			Topic:             topic,
			NumPartitions:     t.options.TopicPartition,
			ReplicationFactor: t.options.TopicReplicationFactor,
		})
	}

	return conn.CreateTopics(configs...)
}

// checkTopic panics if the topic is already registered
func (t Transport) checkTopic(topic string) {
	if _, ok := t.handlers[topic]; ok {
		log.Fatalf("Handler for topic %s is already registered", topic)
	}
	if _, ok := t.rawMsgHandlers[topic]; ok {
		log.Fatalf("Handler for topic %s is already registered", topic)
	}
}

func normalizeTopic(topic string) string {
	return strings.Replace(topic, ":", "_", -1)
}
