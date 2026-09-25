-- +goose Up
-- Счета

CREATE TABLE accounts (
    id             uuid        PRIMARY KEY DEFAULT uuidv7(),
    currency       char(3)     NOT NULL,
    balance        bigint      NOT NULL DEFAULT 0,
    allow_negative boolean     NOT NULL DEFAULT false,
    created_at     timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT accounts_balance_non_negative CHECK (allow_negative OR balance >= 0)
);

-- Транзакции
CREATE TABLE transactions (
    id              uuid        PRIMARY KEY DEFAULT uuidv7(),
    idempotency_key text        NOT NULL,
    description     text        NOT NULL DEFAULT '',
    created_at      timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT transactions_idempotency_key_uniq UNIQUE (idempotency_key)
);
-- Журнал проводок
CREATE TABLE entries (
    id             bigint      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    transaction_id uuid        NOT NULL REFERENCES transactions (id),
    account_id     uuid        NOT NULL REFERENCES accounts (id),
    amount         bigint      NOT NULL CHECK (amount <> 0),
    created_at     timestamptz NOT NULL DEFAULT now()
);

-- выписка по счёту - все проводки счёта X по порядку
CREATE INDEX entries_account_id_id_idx ON entries (account_id, id);
-- все проводки конкретной транзакции
CREATE INDEX entries_transaction_id_idx ON entries (transaction_id);

-- +goose Down
DROP TABLE entries;
DROP TABLE transactions;
DROP TABLE accounts;