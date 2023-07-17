package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"

	"github.com/ansakharov/lets_test/cmd/config"
	create_order_handler "github.com/ansakharov/lets_test/handler/create_order"
	echo_handler "github.com/ansakharov/lets_test/handler/echo"
	get_orders_handler "github.com/ansakharov/lets_test/handler/get_orders"
	"github.com/ansakharov/lets_test/internal/kafka"
	orderRepo "github.com/ansakharov/lets_test/internal/pkg/repository/order"
	orderUCase "github.com/ansakharov/lets_test/internal/usecase/order"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"

	"github.com/ansakharov/lets_test/logger"
	_ "github.com/jackc/pgx/v4/pgxpool"
	"github.com/sirupsen/logrus"
)

const (
	echoRoute   = "/echo"
	orderRoute  = "/order"
	ordersRoute = "/orders"
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

	r := mux.NewRouter()

	// echo
	r.HandleFunc(echoRoute, echo_handler.Handler("Your message: ").ServeHTTP).Methods("GET")

	pool, err := pgxpool.Connect(context.Background(), cfg.DbConnString)
	if err != nil {
		return fmt.Errorf("can't create pg pool: %s", err.Error())
	}
	repo := orderRepo.New(pool)
	orderUseCase := orderUCase.New(repo, kafka.NewProducer(cfg.KafkaPort))

	createOrderHandleFunc := create_order_handler.New(orderUseCase, log).Create(ctx).ServeHTTP
	// create order
	r.HandleFunc(orderRoute, createOrderHandleFunc).Methods("POST")

	getOrderHandlerFunc := get_orders_handler.New(orderUseCase, log).Get(ctx).ServeHTTP
	// get orders
	r.HandleFunc(ordersRoute, getOrderHandlerFunc).Methods("GET")

	if err != nil {
		return fmt.Errorf("can't init router: %s", err.Error())
	}

	log.Print("The service is ready to listen and serve.")

	return errors.Wrap(http.ListenAndServe(
		cfg.AppPort,
		r,
	), "http.ListenAndServe")
}
