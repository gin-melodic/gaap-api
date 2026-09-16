-- Exchange rates for base-currency valuation and multi-currency support.
-- Rates are anchored to a single base currency (default USD): each row stores
-- "1 base_currency = rate quote_currency". Cross-rates are derived at runtime
-- by dividing two anchor rates, so we do not store N x N pairs.
--
-- source distinguishes a daily reference rate ('reference') from a manual
-- override ('manual'). Reference syncs must never overwrite 'manual' rows.
CREATE TABLE IF NOT EXISTS exchange_rates (
    base_currency VARCHAR(10) NOT NULL,
    quote_currency VARCHAR(10) NOT NULL,
    rate NUMERIC(28, 9) NOT NULL,
    source VARCHAR(16) NOT NULL DEFAULT 'reference',
    fetched_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (base_currency, quote_currency)
);

CREATE INDEX IF NOT EXISTS idx_exchange_rates_quote_currency ON exchange_rates(quote_currency);
