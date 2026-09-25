-- +goose Up
CREATE TABLE currencies (
    code        char(3)  PRIMARY KEY,
    minor_units smallint NOT NULL CHECK (minor_units BETWEEN 0 AND 4)
);

INSERT INTO currencies (code, minor_units) VALUES
    ('RUB', 2),
    ('USD', 2),
    ('EUR', 2);

ALTER TABLE accounts
    ADD CONSTRAINT accounts_currency_fkey
    FOREIGN KEY (currency) REFERENCES currencies (code);

-- +goose Down
ALTER TABLE accounts DROP CONSTRAINT accounts_currency_fkey;
DROP TABLE currencies;