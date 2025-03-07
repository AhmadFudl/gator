package main

import (
	"context"
	"database/sql"
	"encoding/xml"
	"fmt"
	"html"
	"net/http"
	"os"
	"time"

	"github.com/ahmadfudl/gator/internal/database"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type (
	Feed struct {
		Channel Channel `xml:"channel"`
	}
	Channel struct {
		Title       string `xml:"title"`
		Link        Link   `xml:"link"`
		Description string `xml:"description"`
		Items       []Item `xml:"item"`
	}
	Link struct {
		Href string `xml:"href,attr"`
	}
	Item struct {
		Title       string `xml:"title"`
		Link        string `xml:"link"`
		Description string `xml:"description"`
		PubDate     string `xml:"pubDate"`
	}
)

func fetchFeed(ctx context.Context, url string) (*Feed, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("rss: %w", err)
	}
	req.Header.Set("user-agent", "gator")

	c := &http.Client{
		Timeout: 5 * time.Minute,
	}
	res, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("rss: %w", err)
	}
	defer res.Body.Close()

	feed := &Feed{}
	dec := xml.NewDecoder(res.Body)
	if err := dec.Decode(feed); err != nil {
		return nil, fmt.Errorf("rss: %w", err)
	}
	feed.html_unescape_feed()

	return feed, nil
}

func (f *Feed) html_unescape_feed() {
	f.Channel.Title = html.UnescapeString(f.Channel.Title)
	f.Channel.Description = html.UnescapeString(f.Channel.Description)
	for i := range f.Channel.Items {
		f.Channel.Items[i].Title = html.UnescapeString(f.Channel.Items[i].Title)
		f.Channel.Items[i].Description = html.UnescapeString(f.Channel.Items[i].Description)
	}
}

func scrapeFeeds(s *state) {
	f, err := s.db.GetNextFeed(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "gator: %v", err)
		return
	}

	err = s.db.MarkFeedFetched(context.Background(),
		database.MarkFeedFetchedParams{
			LastFetchedAt: sql.NullTime{Time: time.Now(), Valid: true},
			UpdatedAt:     time.Now(),
			ID:            f.ID,
		})
	if err != nil {
		fmt.Fprintf(os.Stderr, "gator: %v", err)
		return
	}

	feed, err := fetchFeed(context.Background(), f.Url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gator: %v", err)
		return
	}

	items := feed.Channel.Items
	for i := range items {
		cp := database.CreatePostParams{
			ID:          uuid.New(),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Title:       items[i].Title,
			Url:         items[i].Link,
			Description: sql.NullString{String: items[i].Description},
			FeedID:      f.ID,
		}
		if items[i].Description != "" {
			cp.Description.Valid = true
		}

		layout := "Mon, 02 Jan 2006 15:04:05 -0700"
		pubdata, err := time.Parse(layout, items[i].PubDate)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gator %v", err)
		} else {
			cp.PublishedAt = sql.NullTime{Time: pubdata, Valid: true}
		}

		err = s.db.CreatePost(context.Background(), cp)
		if err != nil {
			// error code for unique constraint vioaltion
			// 23505 unique_violation
			// https://www.postgresql.org/docs/9.3/errcodes-appendix.html
			if err, ok := err.(*pq.Error); ok && err.Code.Class() == "23505" {
				return
			}
			fmt.Fprintf(os.Stderr, "gator %v", err)
		}
	}

	fmt.Printf("channel:\n\ttitle: %s\n\tlink: %s\n",
		feed.Channel.Title, feed.Channel.Link.Href)

}
