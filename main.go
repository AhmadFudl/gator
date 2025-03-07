package main

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/ahmadfudl/gator/internal/config"
	"github.com/ahmadfudl/gator/internal/database"
	_ "github.com/lib/pq"
)

type state struct {
	cfg *config.Config
	db  *database.Queries
}

func main() {
	cfg, err := config.Read()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gator: %v\n", err)
		os.Exit(1)
	}

	db, err := sql.Open("postgres", cfg.Db_url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gator: %v\n", err)
		os.Exit(1)
	}

	dbqs := database.New(db)
	s := state{cfg: cfg, db: dbqs}

	c := commands{
		make(map[string]handler),
	}
	c.register("login", handler{
		d: "set the current user",
		f: _login,
	})
	c.register("register", handler{
		d: "register new user",
		f: _register,
	})
	c.register("reset", handler{
		d: "reset all database records",
		f: _reset,
	})
	c.register("users", handler{
		d: "list users",
		f: _users,
	})
	c.register("agg", handler{
		d: "fetch rss feed",
		f: _agg,
	})
	c.register("addfeed", handler{
		d: "add new feed",
		f: _addfeed,
	})
	c.register("feeds", handler{
		d: "list feeds",
		f: _feeds,
	})
	c.register("follow", handler{
		d: "follow feed",
		f: _follow,
	})
	c.register("unfollow", handler{
		d: "unfollow feed",
		f: _unfollow,
	})
	c.register("following", handler{
		d: "list followed feeds",
		f: _following,
	})
	c.register("browse", handler{
		d: "view all posts from the feeds the user follows",
		f: _browse,
	})

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: gator <command> [<args>]\n\n")
		fmt.Fprintf(os.Stderr, "Commands:\n")
		for k, v := range c.m {
			// 10 the longest command
			fmt.Fprintf(os.Stderr, "\t%-10s %s\n", k, v.d)
		}
		os.Exit(1)
	}

	cmd := command{}
	cmd.name = os.Args[1]
	cmd.args = os.Args[2:]

	err = c.run(&s, cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
