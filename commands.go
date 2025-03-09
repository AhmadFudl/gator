package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ahmadfudl/gator/internal/database"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type (
	command struct {
		name string
		args []string
	}
	commands struct {
		m map[string]handler
	}
	handler struct {
		f func(*state, command) error
		d string
	}
)

func (c *commands) register(name string, h handler) {
	c.m[name] = h
}

func (c *commands) run(s *state, cmd command) error {
	if handler, ok := c.m[cmd.name]; !ok {
		return fmt.Errorf(
			"gator: '%s' is not a gator command. See gator help.",
			cmd.name,
		)
	} else {
		return handler.f(s, cmd)
	}
}

func _migrate(s *state, cmd command) error {
	command := "up"
	if len(cmd.args) > 1 {
		return fmt.Errorf(`fatal: You must provide only a command for %s.

Usage: gator %[1]s [<up> | <reset>]`,
			cmd.name)
	} else if len(cmd.args) == 1 {
		if cmd.args[0] != "up" && cmd.args[0] != "reset" {
			return fmt.Errorf(`fatal: Unknow command '%s'.

Usage: gator %s [<up> | <reset>]`,
				cmd.args[0], cmd.name)
		}
		command = cmd.args[0]
	}

	var err error
	if command == "up" {
		_, err = s.prov.Up(context.Background())
	} else {
		_, err = s.prov.DownTo(context.Background(), 0)
	}
	if err != nil {
		return fmt.Errorf("gator: %v", err)
	}

	return nil
}

func _login(s *state, cmd command) error {
	if len(cmd.args) > 1 {
		return fmt.Errorf(`fatal: You must provide only a username for %s.

Usage: gator %[1]s <username>`,
			cmd.name)
	} else if len(cmd.args) < 1 {
		return fmt.Errorf(`fatal: You must provide a username for %s.

Usage: gator %[1]s <username>`,
			cmd.name)
	}

	username := strings.ToLower(cmd.args[0])
	u, err := s.db.GetUser(context.Background(), username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf(`fatal: User '%s' not registered.

Usage: gator register '%[1]s'`,
				cmd.args[0])
		}
		return fmt.Errorf("gator: %w", err)
	}

	err = s.cfg.SetUser(u.Name)
	if err != nil {
		return fmt.Errorf("gator: %w", err)
	}
	fmt.Printf("gator: User '%s' has been set.\n", cmd.args[0])

	return nil
}

func _register(s *state, cmd command) error {
	if len(cmd.args) > 1 {
		return fmt.Errorf(`fatal: You must provide only a username for %s.

Usage: gator %[1]s <username>`,
			cmd.name)
	} else if len(cmd.args) < 1 {
		return fmt.Errorf(`fatal: You must provide a username for %s.

Usage: gator %[1]s <username>`,
			cmd.name)
	}

	username := strings.ToLower(cmd.args[0])
	u, err := s.db.CreateUser(context.Background(),
		database.CreateUserParams{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Name:      username,
		})
	if err != nil {
		// error code for unique constraint vioaltion
		// 23505 unique_violation
		// https://www.postgresql.org/docs/9.3/errcodes-appendix.html
		if err, ok := err.(*pq.Error); ok && err.Code == "23505" {
			return fmt.Errorf(
				"fatal: User '%s' is already registered.",
				username)
		}
		return fmt.Errorf("gator: %w", err)
	}

	err = s.cfg.SetUser(u.Name)
	if err != nil {
		return fmt.Errorf("gator: %w", err)
	}

	fmt.Printf("gator: User '%s' was created successfully\n", u.Name)
	// debug print
	fmt.Println(u)

	return nil
}

func _reset(s *state, cmd command) error {
	if len(cmd.args) != 0 {
		return fmt.Errorf(`fatal: Too mangy args.

Usage: gator %s`,
			cmd.name)
	}

	if err := s.db.DeleteUsers(context.Background()); err != nil {
		return fmt.Errorf("gator: %w", err)
	}

	if err := s.cfg.SetUser(""); err != nil {
		return fmt.Errorf("gator: %w", err)
	}
	return nil
}

func _users(s *state, cmd command) error {
	if len(cmd.args) != 0 {
		return fmt.Errorf(`fatal: Too mangy args.

Usage: gator %s`,
			cmd.name)
	}

	us, err := s.db.GetUsers(context.Background())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("gator: %w", err)
	}

	for _, u := range us {
		fmt.Printf("* %s", u.Name)
		if u.Name == s.cfg.Current_user_name {
			fmt.Printf(" (current)")
		}
		fmt.Println()
	}

	return nil
}

func _agg(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf(`fatal: You must provide a between-requests' time for %s.

Usage: gator %[1]s <time_between_requests>`,
			cmd.name)
	}
	time_between_req, err := time.ParseDuration(cmd.args[0])
	if err != nil {
		return fmt.Errorf("gator: %w", err)
	}
	fmt.Println("Collecting feeds every", time_between_req)

	ticker := time.NewTicker(time_between_req)
	for ; ; <-ticker.C {
		scrapeFeeds(s)
	}
}

func _addfeed(s *state, cmd command) error {
	if len(cmd.args) > 2 {
		return fmt.Errorf(`fatal: You must provide only a name and a url for %s.

Usage: gator %[1]s <name> <url>`,
			cmd.name)
	} else if len(cmd.args) < 2 {
		return fmt.Errorf(`fatal: You must provide a name and a url for %s.

Usage: gator %[1]s <name> <url>`,
			cmd.name)
	}
	// validate the feed url and name
	feed_name, feed_url := cmd.args[0], cmd.args[1]
	if feed_name == "" {
		return fmt.Errorf("gator: Can't have empty feed name.")
	}
	if feed_url == "" {
		return fmt.Errorf("gator: Can't have empty feed url.")
	}

	u, err := s.db.GetUser(context.Background(), s.cfg.Current_user_name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf(`fatal: Login first to add feeds.

Usage: gator login <username>`)
		}
		return fmt.Errorf("gator: %w", err)
	}

	feed, err := s.db.CreateFeed(context.Background(), database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      feed_name,
		Url:       feed_url,
		UserID:    u.ID,
	})
	if err != nil {
		// error code for foreign key constraint vioaltion
		// 23503 foreign_key_violation
		// https://www.postgresql.org/docs/9.3/errcodes-appendix.html
		// TODO: check if we need to handle this error code
		return fmt.Errorf("gator: %w", err)
	}

	_, err = s.db.CreateFeedFollow(context.Background(),
		database.CreateFeedFollowParams{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			FeedID:    feed.ID,
			UserID:    u.ID,
		})
	if err != nil {
		// error code for unique constraint vioaltion
		// 23505 unique_violation
		// https://www.postgresql.org/docs/9.3/errcodes-appendix.html
		if err, ok := err.(*pq.Error); ok && err.Code == "23505" {
			return fmt.Errorf("fatal: You can't follow the same feed twice.")
		}
		return fmt.Errorf("gator: %w", err)
	}

	fmt.Printf("gator: Feed '%s' was added successfully\n", feed.Name)
	// debug print
	fmt.Println(feed)

	return nil
}

func _feeds(s *state, cmd command) error {
	if len(cmd.args) != 0 {
		return fmt.Errorf(`fatal: Too mangy args.

Usage: gator %s`,
			cmd.name)
	}

	feeds, err := s.db.GetFeeds(context.Background())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("gator: %w", err)
	}

	for _, feed := range feeds {
		fmt.Printf("feed name    : %s\nfeed url     : %s\nfeed creator : %s\n\n",
			feed.Name, feed.Url, feed.Creator)
	}

	return nil
}

func _follow(s *state, cmd command) error {
	if len(cmd.args) > 1 {
		return fmt.Errorf(`fatal: You must provide only a url for %s.

Usage: gator %[1]s <url>`,
			cmd.name)
	} else if len(cmd.args) < 1 {
		return fmt.Errorf(`fatal: You must provide a url for %s.

Usage: gator %[1]s <username>`,
			cmd.name)
	}

	url := cmd.args[0]
	if url == "" {
		return fmt.Errorf("gator: Can't have empty url.")
	}

	u, err := s.db.GetUser(context.Background(), s.cfg.Current_user_name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf(`fatal: Login first to follow a feed.

Usage: gator login <username>`)
		}
		return fmt.Errorf("gator: %w", err)
	}

	feed, err := s.db.GetFeed(context.Background(), url)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf(`fatal: Feed doesn't exist.

Usage: gator addfeed <name> <url>`)
		}
		return fmt.Errorf("gator: %w", err)
	}

	ff, err := s.db.CreateFeedFollow(context.Background(),
		database.CreateFeedFollowParams{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			FeedID:    feed.ID,
			UserID:    u.ID,
		})
	if err != nil {
		// error code for unique constraint vioaltion
		// 23505 unique_violation
		// https://www.postgresql.org/docs/9.3/errcodes-appendix.html
		if err, ok := err.(*pq.Error); ok && err.Code == "23505" {
			return fmt.Errorf("fatal: You can't follow the same feed twice.")
		}
		return fmt.Errorf("gator: %w", err)
	}

	fmt.Println("Done.")
	fmt.Println("\tfeed:", ff.Feed)
	fmt.Println("\tuser:", ff.User)

	return nil
}

func _unfollow(s *state, cmd command) error {
	if len(cmd.args) > 1 {
		return fmt.Errorf(`fatal: You must provide only a url for %s.

Usage: gator %[1]s <url>`,
			cmd.name)
	} else if len(cmd.args) < 1 {
		return fmt.Errorf(`fatal: You must provide a url for %s.

Usage: gator %[1]s <username>`,
			cmd.name)
	}

	url := cmd.args[0]
	if url == "" {
		return fmt.Errorf("gator: Can't have empty url.")
	}

	u, err := s.db.GetUser(context.Background(), s.cfg.Current_user_name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf(`fatal: Login first to unfollow a feed.

Usage: gator login <username>`)
		}
		return fmt.Errorf("gator: %w", err)
	}

	feed, err := s.db.GetFeed(context.Background(), url)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf(`fatal: feed doesn't exist.

Usage: gator addfeed <name> <url>`)
		}
		return fmt.Errorf("gator: %w", err)
	}

	err = s.db.DeleteFeedFollow(context.Background(),
		database.DeleteFeedFollowParams{
			UserID: u.ID,
			FeedID: feed.ID,
		})
	if err != nil {
		return fmt.Errorf("gator: %w", err)
	}

	fmt.Println("Done.")

	return nil
}

func _following(s *state, cmd command) error {
	if len(cmd.args) != 0 {
		return fmt.Errorf(`fatal: Too mangy args.

Usage: gator %s`,
			cmd.name)
	}

	u, err := s.db.GetUser(context.Background(), s.cfg.Current_user_name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf(`fatal: Login first to see follow list.

Usage: gator login <username>`)
		}
		return fmt.Errorf("gator: %w", err)
	}

	ffs, err := s.db.GetFeedFollowsForUser(context.Background(), u.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("gator: %w", err)
	}

	for _, ff := range ffs {
		fmt.Printf("feed: %s\nurl:  %s\n\n", ff.Feed, ff.Url)
	}

	return nil
}

func _browse(s *state, cmd command) error {
	limit := 2
	if len(cmd.args) > 1 {
		return fmt.Errorf(`fatal: You must provide only a limit for %s.

Usage: gator %[1]s <url>`,
			cmd.name)
	} else if len(cmd.args) == 1 {
		int, err := strconv.Atoi(cmd.args[0])
		if err != nil {
			return fmt.Errorf(`fatal: You must provide a limit for %s.

Usage: gator %[1]s <limit>`,
				cmd.name)
		}
		limit = int
	}

	u, err := s.db.GetUser(context.Background(), s.cfg.Current_user_name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf(`fatal: Login first to browse feeds.

Usage: gator login <username>`)
		}
		return fmt.Errorf("gator: %w", err)
	}

	posts, err := s.db.GetPostsUser(context.Background(),
		database.GetPostsUserParams{
			UserID: u.ID,
			Limit:  int32(limit),
		})

	for i := range posts {
		fmt.Printf(`
post:
	title:       %s
	link:        %s
	pubDate:     %v
	description: %s
`,
			posts[i].Title, posts[i].Url,
			posts[i].PublishedAt.Time, posts[i].Description.String)
	}

	return nil
}
