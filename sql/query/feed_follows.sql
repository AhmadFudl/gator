-- name: CreateFeedFollow :one
WITH
	new_feed_follow AS (
		INSERT INTO
			feed_follows (id, created_at, updated_at, feed_id, user_id)
		VALUES
			($1, $2, $3, $4, $5)
		RETURNING
			*
	)
SELECT
	new_feed_follow.*,
	users.name AS user,
	feeds.name AS feed
FROM
	new_feed_follow
	INNER JOIN users ON new_feed_follow.user_id = users.id
	INNER JOIN feeds ON new_feed_follow.feed_id = feeds.id
;

-- name: GetFeedFollowsForUser :many
SELECT
	feed_follows.*,
	feeds.name AS feed,
	feeds.url  AS url
FROM
	feed_follows
	INNER JOIN feeds ON feed_follows.feed_id = feeds.id
WHERE
	feed_follows.user_id = $1
;

-- name: DeleteFeedFollow :exec
DELETE FROM feed_follows
WHERE
	user_id = $1 AND feed_id = $2
;
