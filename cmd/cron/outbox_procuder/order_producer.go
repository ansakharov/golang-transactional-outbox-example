package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/ansakharov/lets_test/cmd/config"
	"github.com/ansakharov/lets_test/internal/cron/outbox_producer"
	"github.com/ansakharov/lets_test/internal/kafka"
	"github.com/ansakharov/lets_test/internal/pkg/repository/order"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"

	"github.com/ansakharov/lets_test/logger"
	_ "github.com/jackc/pgx/v4/pgxpool"
	"github.com/sirupsen/logrus"
)

func main() {
	// Get logger interface.
	l := logger.New()

	if err := mainNoExit(l); err != nil {
		l.Fatalf("fatal err: %s", err.Error())
	}
}

func mainNoExit(log logrus.FieldLogger) error {
	// get application cfg
	confFlag := flag.String("conf", "", "cfg yaml file")
	flag.Parse()

	confString := *confFlag
	if confString == "" {
		return fmt.Errorf(" 'conf' flag required")
	}

	cfg, err := config.Parse(confString)
	if err != nil {
		return errors.Wrap(err, "config.Parse")
	}

	log.Println(cfg)
	log.Println("Starting the service...")

	ctx := context.Background()

	producer := kafka.NewProducer(cfg.KafkaPort)

	pool, err := pgxpool.Connect(context.Background(), cfg.DbConnString)
	if err != nil {
		return fmt.Errorf("can't create pg pool: %s", err.Error())
	}

	outboxProducer := outbox_producer.New(producer, pool, order.OutboxTable, log)

	return errors.Wrap(outboxProducer.ProduceMessages(ctx), "outboxProducer.ProduceMessages")
}
