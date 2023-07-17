package order

import (
	"context"
	baseErr "errors"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	sq "github.com/Masterminds/squirrel"
	"github.com/ansakharov/lets_test/internal/pkg/entity/order"
	order_entity "github.com/ansakharov/lets_test/internal/pkg/entity/order"
	"github.com/hashicorp/go-uuid"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	// tables
	ordersTable     = "orders"
	itemsTable      = "items"
	orderItemsTable = "order_items"

	OutboxTable = "outbox"
)

type Repository struct {
	db *pgxpool.Pool
}

//go:generate ${MOQPATH}moq -skip-ensure -pkg mocks -out ./mocks2/repo_mock.go . OrderRepo
type OrderRepo interface {
	Save(ctx context.Context, log logrus.FieldLogger, order *order_entity.Order) (uint64, error)
	Get(ctx context.Context, log logrus.FieldLogger, IDs []uint64) (map[uint64]order.Order, error)
}

// New instance of repository.
func New(pool *pgxpool.Pool) *Repository {
	return &Repository{db: pool}
}

// Save new order to DB.
func (r *Repository) Save(ctx context.Context, log logrus.FieldLogger, order *order_entity.Order) (uint64, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("can't create tx: %s", err.Error())
	}

	defer func() {
		if err != nil {
			rollbackErr := tx.Rollback(ctx)
			if rollbackErr != nil {
				err = baseErr.Join(err, rollbackErr)
			}
		}
	}()

	query, args, err := sq.
		Insert(ordersTable).
		Columns("user_id", "payment_type", "created_at").
		Values(
			order.UserID,
			order.PaymentType,
			time.Now().Format(time.RFC3339),
		).
		Suffix("RETURNING id").
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("can't build sql: %s", err.Error())
	}

	// insert into orders table.
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("trx err: %w", err)
	}
	defer rows.Close()

	var orderID uint64
	for rows.Next() {
		if scanErr := rows.Scan(&orderID); scanErr != nil {
			return 0, fmt.Errorf("can't scan orderID: %s", scanErr.Error())
		}
	}

	builder := sq.
		Insert(orderItemsTable).
		Columns(
			"order_id",
			"item_id",
			"original_amount",
			"discounted_amount",
		)

	for _, service := range order.Items {
		builder = builder.Values(
			orderID,
			service.ID,
			service.Amount,
			service.DiscountedAmount)
	}

	query, args, err = builder.PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return 0, fmt.Errorf("trx: %w", err)
	}

	// insert into services table.
	_, err = tx.Exec(ctx, query, args...)
	if err != nil {
		return 0, errors.Wrap(err, "txErr")
	}

	eventID, err := uuid.GenerateUUID()
	if err != nil {
		return 0, err
	}

	query, args, err = sq.
		Insert(OutboxTable).
		Columns("event_id", "order_id").
		Values(
			eventID,
			orderID,
		).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("can't build sql: %s", err.Error())
	}

	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return 0, fmt.Errorf("tx.Exec: %s", err.Error())
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("can't commit tx: %s", err.Error())
	}

	return orderID, nil
}

// Get returns map of orders.
func (r *Repository) Get(ctx context.Context, log logrus.FieldLogger, IDs []uint64) (map[uint64]order.Order, error) {
	ordersMap := make(map[uint64]order.Order, len(IDs))
	or := sq.Or{}
	orOrderItems := sq.Or{}
	for _, id := range IDs {
		or = append(or, sq.Eq{"id": id})
		orOrderItems = append(orOrderItems, sq.Eq{"order_id": id})
	}

	// build query.
	query, args, err := sq.
		Select("id", "user_id", "payment_type").
		From(ordersTable).
		Where(or).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("can't build query: %s", err.Error())
	}

	// get orders.
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("can't select orders: %s", err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		ord := order.Order{}
		scanErr := rows.Scan(&ord.ID, &ord.UserID, &ord.PaymentType)
		if scanErr != nil {
			return nil, fmt.Errorf("can't scan order: %s", scanErr.Error())
		}
		ordersMap[ord.ID] = ord
	}

	// build query
	query, args, err = squirrel.
		Select("order_id", "item_id", "original_amount", "discounted_amount").
		From(orderItemsTable).
		Where(orOrderItems).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("can't build query")
	}

	// get order items
	rows, err = r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("can't select order_items: %s", err.Error())
	}
	defer rows.Close()

	// put items in orders.
	for rows.Next() {
		service := order.Item{}
		err = rows.Scan(&service.OrderID, &service.ID, &service.Amount, &service.DiscountedAmount)

		if err != nil {
			return nil, fmt.Errorf("can't scan order: %s", err.Error())
		}

		ord := ordersMap[service.OrderID]
		ord.Items = append(ord.Items, service)

		ordersMap[service.OrderID] = ord
	}
	return ordersMap, nil
}
