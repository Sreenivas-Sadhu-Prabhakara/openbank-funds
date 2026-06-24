-- Funds (Confirmation of Funds, CBPII) service schema. Owned exclusively by the
-- funds microservice; no other service reads or writes these tables.
CREATE SCHEMA IF NOT EXISTS funds;

CREATE TABLE IF NOT EXISTS funds.funds_confirmations (
    id                  TEXT PRIMARY KEY,
    consent_id          TEXT        NOT NULL,
    creation_dt         TIMESTAMPTZ NOT NULL,
    reference           TEXT,

    -- The amount whose availability was checked.
    instructed_amount   TEXT        NOT NULL,
    instructed_currency TEXT        NOT NULL,

    -- Outcome of the availability check and the instant it was determined.
    funds_available     BOOLEAN     NOT NULL,
    funds_available_dt  TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_funds_confirmations_consent ON funds.funds_confirmations (consent_id);
