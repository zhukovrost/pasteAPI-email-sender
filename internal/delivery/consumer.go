package delivery

import (
	"encoding/json"
	"fmt"
	"github.com/zhukovrost/pasteAPI-email-sender/internal/models"
	"os"
	"os/signal"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/zhukovrost/pasteAPI-email-sender/internal/service"
)

type Config struct {
	URL          string
	Exchange     string
	ExchangeType string
	Queue        string
	BindingKey   string
	ConsumerTag  string
}

func New(cfg Config, lifetime *time.Duration, emailService *service.Mailer) {
	c, err := NewConsumer(cfg.URL, cfg.Exchange, cfg.ExchangeType, cfg.Queue, cfg.BindingKey, cfg.ConsumerTag, emailService)
	if err != nil {
		emailService.Logger.Fatalf("%s", err)
	}

	SetupCloseHandler(c)

	if *lifetime > 0 {
		emailService.Logger.Printf("running for %s", *lifetime)
		time.Sleep(*lifetime)
	} else {
		emailService.Logger.Printf("running until Consumer is done")
		<-c.done
	}

	emailService.Logger.Printf("shutting down")

	if err := c.Shutdown(); err != nil {
		emailService.Logger.Fatalf("error during shutdown: %s", err)
	}
}

type Consumer struct {
	conn         *amqp.Connection
	channel      *amqp.Channel
	tag          string
	done         chan error
	emailService *service.Mailer
	shutdown     chan struct{}
}

func SetupCloseHandler(consumer *Consumer) {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		consumer.emailService.Logger.Info("rabbitmq stopped")
		if err := consumer.Shutdown(); err != nil {
			consumer.emailService.Logger.Fatalf("error during shutdown: %s", err)
		}
		os.Exit(0)
	}()
}

func NewConsumer(amqpURI, exchange, exchangeType, queueName, key, ctag string, emailService *service.Mailer) (*Consumer, error) {
	c := &Consumer{
		conn:         nil,
		channel:      nil,
		tag:          ctag,
		done:         make(chan error),
		emailService: emailService,
		shutdown:     make(chan struct{}),
	}

	var err error

	config := amqp.Config{Properties: amqp.NewConnectionProperties()}
	config.Properties.SetClientConnectionName(c.tag)
	emailService.Logger.Debug("dialing")
	c.conn, err = amqp.DialConfig(amqpURI, config)
	if err != nil {
		return nil, fmt.Errorf("dial: %s", err)
	}

	go func() {
		emailService.Logger.Infof("closing: %s", <-c.conn.NotifyClose(make(chan *amqp.Error)))
		close(c.shutdown)
	}()

	emailService.Logger.Infof("got Connection, getting Channel")
	c.channel, err = c.conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("channel: %s", err)
	}

	emailService.Logger.Infof("got Channel, declaring Exchange (%q)", exchange)
	if err = c.channel.ExchangeDeclare(
		exchange,     // name of the exchange
		exchangeType, // type
		true,         // durable
		false,        // delete when complete
		false,        // internal
		false,        // noWait
		nil,          // arguments
	); err != nil {
		return nil, fmt.Errorf("exchange Declare: %s", err)
	}

	emailService.Logger.Infof("declared Exchange, declaring Queue %q", queueName)
	queue, err := c.channel.QueueDeclare(
		queueName, // name of the queue
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // noWait
		nil,       // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("queue Declare: %s", err)
	}

	emailService.Logger.Infof("declared Queue (%q %d messages, %d consumers), binding to Exchange (key %q)",
		queue.Name, queue.Messages, queue.Consumers, key)

	if err = c.channel.QueueBind(
		queue.Name, // name of the queue
		key,        // bindingKey
		exchange,   // sourceExchange
		false,      // noWait
		nil,        // arguments
	); err != nil {
		return nil, fmt.Errorf("queue Bind: %s", err)
	}

	emailService.Logger.Infof("Queue bound to Exchange, starting Consume (consumer tag %q)", c.tag)
	deliveries, err := c.channel.Consume(
		queue.Name, // name
		c.tag,      // consumerTag,
		true,       // autoAck
		false,      // exclusive
		false,      // noLocal
		false,      // noWait
		nil,        // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("queue Consume: %s", err)
	}

	go c.Handle(deliveries, c.done)

	return c, nil
}

func (c *Consumer) Shutdown() error {
	select {
	case <-c.shutdown:
		// Already shutting down, return
		return nil
	default:
	}

	if c.channel != nil {
		if err := c.channel.Cancel(c.tag, true); err != nil {
			c.emailService.Logger.Errorf("consumer cancel failed: %s", err)
		}

		if err := c.channel.Close(); err != nil {
			c.emailService.Logger.Errorf("AMQP channel close error: %s", err)
		}
	}

	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			c.emailService.Logger.Errorf("AMQP connection close error: %s", err)
		}
	}

	c.emailService.Logger.Infof("AMQP shutdown OK")
	return <-c.done
}

func (c *Consumer) Handle(deliveries <-chan amqp.Delivery, done chan error) {
	cleanup := func() {
		c.emailService.Logger.Infof("handle: deliveries channel closed")
		close(done)
	}

	defer cleanup()

	for d := range deliveries {
		c.emailService.Logger.Debugf("got new delivery: [%v]", d.DeliveryTag)

		var msg models.Email

		err := json.Unmarshal(d.Body, &msg)
		if err != nil {
			c.emailService.Logger.Errorf("error decoding JSON: %v", err)
			continue
		}
		err = c.emailService.SendEmail(&msg)
		if err != nil {
			c.emailService.Logger.Errorf("error sending email: %v", err)
			continue
		}

		c.emailService.Logger.Debug("successfully sent email")
	}
}
