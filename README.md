### What it is 
Example of transactional outbox pattern in golang: write new orders with api server, send them to kafka via cron.

More abort pattern: https://microservices.io/patterns/data/transactional-outbox.html

### How to run
```
 // to run postgres, kafka
docker-compose up -d

// to run app
go run cmd/main.go --conf=conf.yaml

// to run worker once
go run cmd/cron/outbox_producer/order_producer/main.go --conf=conf.yaml
```
