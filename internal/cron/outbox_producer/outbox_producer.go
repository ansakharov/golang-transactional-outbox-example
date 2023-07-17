package outbox_producer

import (
	"context"
	baseErr "errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/Shopify/sarama"
	"github.com/ansakharov/lets_test/internal/kafka"
	"github.com/ansakharov/lets_test/internal/pkg/repository/order"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/sirupsen/logrus"
)

type OutboxProducer struct {
	producer sarama.SyncProducer
	// TODO replace with interface
	db          *pgxpool.Pool
	outboxTable string
	log         logrus.FieldLogger
}

type outboxMessage struct {
	OrderID int64  `db:"order_id"`
	EventID string `db:"event_id"`
}

func New(producer sarama.SyncProducer, db *pgxpool.Pool, outboxTable string, log logrus.FieldLogger) *OutboxProducer {
	return &OutboxProducer{
		producer:    producer,
		db:          db,
		outboxTable: outboxTable,
		log:         log,
	}
}

// ProduceMessages from outbox table
func (op *OutboxProducer) ProduceMessages(ctx context.Context) (err error) {
	tx, err := op.db.BeginTx(ctx, pgx.TxOptions{})
	// вычитаем данные на отправку
	defer func() {
		if err != nil {
			rollbackErr := tx.Rollback(ctx)
			if rollbackErr != nil {
				err = baseErr.Join(err, rollbackErr)
			}
		}
	}()
	// build query.
	query, args, err := sq.
		Select("event_id", "order_id").
		From(order.OutboxTable).
		Where(sq.Eq{"sent": false}).
		OrderBy("order_id asc").
		Limit(100).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("can't build query: %s", err.Error())
	}

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	//messages := make([]outboxMessage, 0, 100)
	eventIds := []string{}
	saramaMsgs := make([]*sarama.ProducerMessage, 0, 100)

	for rows.Next() {
		msg := outboxMessage{}
		if err := rows.Scan(&msg.EventID, &msg.OrderID); err != nil {
			return err
		}

		//	messages = append(messages, msg)
		saramaMsgs = append(saramaMsgs, &sarama.ProducerMessage{
			Topic: kafka.Topic,
			Value: sarama.StringEncoder(fmt.Sprintf("{\"event_id\": \"%s\", \"order_id\": %d}", msg.EventID, msg.OrderID)),
		})

		eventIds = append(eventIds, msg.EventID)
	}

	// отправим данные

	err = op.producer.SendMessages(saramaMsgs)
	if err != nil {
		return err
	}

	query, args, err = sq.
		Update(op.outboxTable).
		Set("sent", true).
		Where(sq.Eq{"event_id": eventIds}).
		PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return err
	}

	// пометим данным отправленными

	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("tx.Exec: %s", err.Error())
	}

	// коммит
	return tx.Commit(ctx)
}
