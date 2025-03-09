# Gator

An RSS feed aggregator in Go.

## About

**"Gator"**, you know, because **aggreGATOR** 🐊.

Anyhow, it's a CLI tool that allows users to:
- Add RSS feeds from across the internet to be collected
- Store the collected posts in a PostgreSQL database
- Follow and unfollow RSS feeds that other users have added
- View summaries of the aggregated posts in the terminal, with a link to the
full post

Gator is a multi-user CLI application. There's no server (**yet**--other than
the database!), so it's only intended for local use. But just like games in the
'90s andearly 2000s, that doesn't mean we can't have multiplayer functionality
on a single device!

## Prerequisites

You'll need to install:
- PostgreSQL
- Go

### Setup

1. Create a config file (see the [Config](#config) section for details).
2. Create a PostgreSQL database.
3. Set `db_url` in your config file:
    
    ```
    protocol://username:password@host:port/database?sslmode=disable`
    ```
    No need for SSL in local mode.

## Installation

```console
$ go install github.com/ahmadfudl/gator@latest
```

## Commands

```console
$ gator
Usage: gator <command> [<args>]

Commands:
        addfeed    add new feed
        feeds      list feeds
        unfollow   unfollow feed
        following  list followed feeds
        login      set the current user
        register   register new user
        reset      reset all database records
        agg        fetch rss feed
        browse     view all posts from the feeds the user follows
        migrate    migrates db
        users      list users
        follow     follow feed
```

## Usage

1. **Register a user** `gator register fudl`
2. **Add some feeds**  `gator addfeed "Articles on gingerBill" "https://www.gingerbill.org/article/index.xml"`
3. **Start aggregation** (in a separate terminal)
    `gator agg 1m`
   - `m` = minutes  
   - `h` = hours  
   - and so on...
   This updates one feed at a time with the latest posts.

4. **Browse posts from followed feeds**  
    `gator browse 2`
   Lists posts from the feeds you follow, sorted from oldest to newest.

## Config

Gator stores its configuration in a JSON file named `.gatorconfig.json`,

1. The currently logged-in user.
2. PostgreSQL database connection credentials.

```json
{
    "db_url": "postgres://example",
    "current_user_name": "username"
}
```

For a sample configuration file check: [.gatorconfig.sample.json](/.gatorconfig.sample.json)


> [!NOTE]
> There's no user-based authentication (**yet**).
> if someone has the database credentials, they can act as any user.

