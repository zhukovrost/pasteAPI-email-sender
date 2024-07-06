package app

import (
	"flag"
	"github.com/zhukovrost/pasteAPI-email-sender/internal/delivery"
	"github.com/zhukovrost/pasteAPI/pkg/logger"
	"time"

	"github.com/zhukovrost/pasteAPI-email-sender/internal/config"
	"github.com/zhukovrost/pasteAPI-email-sender/internal/service"
)

func Run(cfg *config.Config) {
	lifetime := flag.Duration("lifetime", 0, "Lifetime for the consumer (0 to run until manually stopped)")

	flag.Parse()
	l := logger.New(cfg.Logger.NeedDebug)

	timeout, err := time.ParseDuration(cfg.RabbitMQ.Timeout)
	if err != nil {
		timeout = time.Second * 5
	}

	l.Debug("Starting email service")
	emailService := service.New(l, service.Config{
		Host:     cfg.SMTP.Host,
		Port:     cfg.SMTP.Port,
		Username: cfg.SMTP.Username,
		Password: cfg.SMTP.Password,
		Sender:   cfg.SMTP.Sender,
		Timeout:  timeout,
	})

	l.Debug("Starting rabbitmq")
	rmqCfg := delivery.Config{
		URL:          cfg.RabbitMQ.URL,
		Exchange:     cfg.RabbitMQ.Exchange,
		ExchangeType: cfg.RabbitMQ.ExchangeType,
		Queue:        cfg.RabbitMQ.Queue,
		BindingKey:   "",
		ConsumerTag:  cfg.RabbitMQ.ConsumerTag,
	}

	delivery.New(rmqCfg, lifetime, emailService)
}
