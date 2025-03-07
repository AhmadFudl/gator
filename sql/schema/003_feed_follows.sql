-- +goose Up
CREATE TABLE feed_follows (
	id         UUID,
	created_at TIMESTAMP NOT NULL,
	updated_at TIMESTAMP NOT NULL,
	feed_id    UUID      NOT NULL,
	user_id    UUID      NOT NULL,
	UNIQUE (feed_id, user_id),
	PRIMARY KEY (id),
	FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
	FOREIGN KEY (feed_id) REFERENCES feeds (id) ON DELETE CASCADE
)
;

-- +goose Down
DROP TABLE feed_follows
;
