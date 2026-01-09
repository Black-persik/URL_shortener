-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS links (
  id           BIGSERIAL PRIMARY KEY,
  code         TEXT NOT NULL UNIQUE,
  original_url TEXT NOT NULL,
  total_clicks BIGINT NOT NULL DEFAULT 0,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS click_events (
  id         BIGSERIAL PRIMARY KEY,
  link_id    BIGINT NOT NULL REFERENCES links(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_click_events_link_id ON click_events(link_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS click_events;
DROP TABLE IF EXISTS links;

-- +goose StatementEnd
