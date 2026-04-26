CREATE UNIQUE INDEX IF NOT EXISTS idx_short_urls_original_url_unique
    ON short_urls (original_url);
