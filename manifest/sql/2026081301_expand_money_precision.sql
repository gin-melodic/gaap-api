-- Preserve the full protobuf int64 units range together with nano precision.
-- NUMERIC(28, 9) provides 19 integer digits and 9 fractional digits.
ALTER TABLE accounts
    ALTER COLUMN balance_decimal TYPE NUMERIC(28, 9);

ALTER TABLE transactions
    ALTER COLUMN balance_decimal TYPE NUMERIC(28, 9);
