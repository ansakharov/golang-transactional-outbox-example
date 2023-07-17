CREATE TABLE outbox (
    id SERIAL PRIMARY KEY,
    event_id TEXT NOT NULL,
    order_id INTEGER NOT NULL,
    sent BOOLEAN DEFAULT FALSE
);
