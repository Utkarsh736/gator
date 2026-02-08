package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Utkarsh736/gator/internal/config"
	"github.com/Utkarsh736/gator/internal/database"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// state holds the application state (config, DB connection)
type state struct {
	db  *database.Queries
	cfg *config.Config
}

// command represents a CLI command with its name and arguments
type command struct {
	name string
	args []string
}

// commands holds all registered command handlers
type commands struct {
	handlers map[string]func(*state, command) error
}

// register adds a new command handler
func (c *commands) register(name string, f func(*state, command) error) {
	c.handlers[name] = f
}

// run executes a command by name if it exists
func (c *commands) run(s *state, cmd command) error {
	handler, exists := c.handlers[cmd.name]
	if !exists {
		return fmt.Errorf("unknown command: %s", cmd.name)
	}
	return handler(s, cmd)
}

// middlewareLoggedIn wraps handlers that require a logged-in user
func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(*state, command) error {
	return func(s *state, cmd command) error {
		// Get current user
		user, err := s.db.GetUser(context.Background(), s.cfg.CurrentUserName)
		if err != nil {
			return fmt.Errorf("couldn't get current user: %w", err)
		}

		// Call the wrapped handler with the user
		return handler(s, cmd, user)
	}
}

// handlerRegister creates a new user
func handlerRegister(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return errors.New("register command requires a username argument")
	}

	name := cmd.args[0]

	// Create user in database
	user, err := s.db.CreateUser(context.Background(), database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      name,
	})

	if err != nil {
		// Check if it's a duplicate key error
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return fmt.Errorf("user %s already exists", name)
		}
		return fmt.Errorf("couldn't create user: %w", err)
	}

	// Set current user in config
	err = s.cfg.SetUser(name)
	if err != nil {
		return fmt.Errorf("couldn't set current user: %w", err)
	}

	fmt.Println("User created successfully:")
	fmt.Printf("  ID: %s\n", user.ID)
	fmt.Printf("  Name: %s\n", user.Name)
	fmt.Printf("  Created at: %s\n", user.CreatedAt)

	return nil
}

// handlerLogin sets the current user in the config
func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return errors.New("login command requires a username argument")
	}

	username := cmd.args[0]

	// Check if user exists in database
	_, err := s.db.GetUser(context.Background(), username)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("user %s doesn't exist", username)
		}
		return fmt.Errorf("couldn't get user: %w", err)
	}

	// Set current user
	err = s.cfg.SetUser(username)
	if err != nil {
		return fmt.Errorf("couldn't set current user: %w", err)
	}

	fmt.Printf("User has been set to: %s\n", username)
	return nil
}

// handlerReset deletes all users from the database
func handlerReset(s *state, cmd command) error {
	err := s.db.DeleteAllUsers(context.Background())
	if err != nil {
		return fmt.Errorf("couldn't reset database: %w", err)
	}

	fmt.Println("Database has been reset successfully")
	return nil
}

// handlerUsers lists all users in the database
func handlerUsers(s *state, cmd command) error {
	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		return fmt.Errorf("couldn't get users: %w", err)
	}

	if len(users) == 0 {
		fmt.Println("No users found")
		return nil
	}

	// Get current user from config
	currentUser := s.cfg.CurrentUserName

	// Print all users
	for _, user := range users {
		if user.Name == currentUser {
			fmt.Printf("* %s (current)\n", user.Name)
		} else {
			fmt.Printf("* %s\n", user.Name)
		}
	}

	return nil
}

// handlerAgg continuously fetches feeds at specified intervals
func handlerAgg(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return errors.New("agg command requires a time_between_reqs argument")
	}

	// Parse duration
	timeBetweenRequests, err := time.ParseDuration(cmd.args[0])
	if err != nil {
		return fmt.Errorf("invalid duration: %w", err)
	}

	fmt.Printf("Collecting feeds every %s\n", timeBetweenRequests)

	// Create ticker
	ticker := time.NewTicker(timeBetweenRequests)
	defer ticker.Stop()

	// Run immediately, then on each tick
	for ; ; <-ticker.C {
		err := scrapeFeeds(s)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error scraping feeds: %v\n", err)
		}
	}
}

// scrapeFeeds fetches the next feed and processes its posts
func scrapeFeeds(s *state) error {
	// Get next feed to fetch
	feed, err := s.db.GetNextFeedToFetch(context.Background())
	if err != nil {
		return fmt.Errorf("couldn't get next feed to fetch: %w", err)
	}

	fmt.Printf("Fetching feed: %s (URL: %s)\n", feed.Name, feed.Url)

	// Mark feed as fetched
	err = s.db.MarkFeedFetched(context.Background(), feed.ID)
	if err != nil {
		return fmt.Errorf("couldn't mark feed as fetched: %w", err)
	}

	// Fetch the RSS feed
	rssFeed, err := fetchFeed(context.Background(), feed.Url)
	if err != nil {
		return fmt.Errorf("couldn't fetch feed: %w", err)
	}

	// Save posts to database
	fmt.Printf("Found %d posts in %s\n", len(rssFeed.Channel.Item), feed.Name)
	for _, item := range rssFeed.Channel.Item {
		// Parse published date - try multiple formats
		var publishedAt sql.NullTime
		if item.PubDate != "" {
			t, err := parsePublishedDate(item.PubDate)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: couldn't parse date %q: %v\n", item.PubDate, err)
			} else {
				publishedAt = sql.NullTime{Time: t, Valid: true}
			}
		}

		// Handle nullable description
		var description sql.NullString
		if item.Description != "" {
			description = sql.NullString{String: item.Description, Valid: true}
		}

		// Create post
		_, err := s.db.CreatePost(context.Background(), database.CreatePostParams{
			ID:          uuid.New(),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Title:       item.Title,
			Url:         item.Link,
			Description: description,
			PublishedAt: publishedAt,
			FeedID:      feed.ID,
		})

		if err != nil {
			// Ignore duplicate URL errors (post already exists)
			if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
				continue
			}
			// Log other errors but don't stop
			fmt.Fprintf(os.Stderr, "Warning: couldn't save post %q: %v\n", item.Title, err)
		}
	}

	fmt.Printf("Saved posts from %s\n\n", feed.Name)
	return nil
}

// parsePublishedDate tries multiple date formats common in RSS feeds
func parsePublishedDate(dateStr string) (time.Time, error) {
	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC822Z,
		time.RFC822,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
	}

	for _, format := range formats {
		t, err := time.Parse(format, dateStr)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("no valid date format found")
}

// handlerAddFeed adds a new feed for the current user
func handlerAddFeed(s *state, cmd command, user database.User) error {
	if len(cmd.args) < 2 {
		return errors.New("addfeed command requires name and url arguments")
	}

	name := cmd.args[0]
	url := cmd.args[1]

	// Create feed (user is already provided)
	feed, err := s.db.CreateFeed(context.Background(), database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      name,
		Url:       url,
		UserID:    user.ID,
	})

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return fmt.Errorf("feed with URL %s already exists", url)
		}
		return fmt.Errorf("couldn't create feed: %w", err)
	}

	// Automatically create feed follow
	_, err = s.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	})

	if err != nil {
		return fmt.Errorf("couldn't follow feed: %w", err)
	}

	fmt.Println("Feed created successfully:")
	fmt.Printf("  ID: %s\n", feed.ID)
	fmt.Printf("  Name: %s\n", feed.Name)
	fmt.Printf("  URL: %s\n", feed.Url)
	fmt.Printf("  User ID: %s\n", feed.UserID)
	fmt.Printf("  Created at: %s\n", feed.CreatedAt)
	fmt.Println("(Automatically followed)")

	return nil
}

// handlerFeeds lists all feeds in the database
func handlerFeeds(s *state, cmd command) error {
	feeds, err := s.db.GetFeeds(context.Background())
	if err != nil {
		return fmt.Errorf("couldn't get feeds: %w", err)
	}

	if len(feeds) == 0 {
		fmt.Println("No feeds found")
		return nil
	}

	fmt.Println("Feeds:")
	for _, feed := range feeds {
		fmt.Printf("* Name: %s\n", feed.Name)
		fmt.Printf("  URL: %s\n", feed.Url)
		fmt.Printf("  User: %s\n", feed.UserName)
		fmt.Println()
	}

	return nil
}

// handlerFollow follows a feed by URL
func handlerFollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) == 0 {
		return errors.New("follow command requires a URL argument")
	}

	url := cmd.args[0]

	// Get feed by URL
	feed, err := s.db.GetFeedByURL(context.Background(), url)
	if err != nil {
		return fmt.Errorf("couldn't find feed: %w", err)
	}

	// Create feed follow
	feedFollow, err := s.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	})

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return fmt.Errorf("already following this feed")
		}
		return fmt.Errorf("couldn't follow feed: %w", err)
	}

	fmt.Printf("%s is now following %s\n", feedFollow.UserName, feedFollow.FeedName)
	return nil
}

// handlerFollowing lists feeds the current user is following
func handlerFollowing(s *state, cmd command, user database.User) error {
	// Get feed follows
	follows, err := s.db.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		return fmt.Errorf("couldn't get feed follows: %w", err)
	}

	if len(follows) == 0 {
		fmt.Println("Not following any feeds")
		return nil
	}

	fmt.Printf("Feeds followed by %s:\n", user.Name)
	for _, follow := range follows {
		fmt.Printf("* %s\n", follow.FeedName)
	}

	return nil
}

// handlerUnfollow unfollows a feed by URL
func handlerUnfollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) == 0 {
		return errors.New("unfollow command requires a URL argument")
	}

	url := cmd.args[0]

	// Get feed by URL
	feed, err := s.db.GetFeedByURL(context.Background(), url)
	if err != nil {
		return fmt.Errorf("couldn't find feed: %w", err)
	}

	// Delete feed follow
	err = s.db.DeleteFeedFollow(context.Background(), database.DeleteFeedFollowParams{
		UserID: user.ID,
		FeedID: feed.ID,
	})

	if err != nil {
		return fmt.Errorf("couldn't unfollow feed: %w", err)
	}

	fmt.Printf("%s has unfollowed %s\n", user.Name, feed.Name)
	return nil
}

// handlerBrowse displays posts from feeds the user follows
func handlerBrowse(s *state, cmd command, user database.User) error {
	limit := 2 // default

	if len(cmd.args) > 0 {
		// Parse limit from args
		var err error
		_, err = fmt.Sscan(cmd.args[0], &limit)
		if err != nil {
			return fmt.Errorf("invalid limit: %w", err)
		}
	}

	posts, err := s.db.GetPostsForUser(context.Background(), database.GetPostsForUserParams{
		UserID: user.ID,
		Limit:  int32(limit),
	})

	if err != nil {
		return fmt.Errorf("couldn't get posts: %w", err)
	}

	if len(posts) == 0 {
		fmt.Println("No posts found. Follow some feeds first!")
		return nil
	}

	fmt.Printf("Found %d posts for %s:\n", len(posts), user.Name)
	fmt.Println(strings.Repeat("=", 80))

	for _, post := range posts {
		fmt.Printf("\nTitle: %s\n", post.Title)
		fmt.Printf("URL: %s\n", post.Url)

		if post.Description.Valid {
			// Truncate long descriptions
			desc := post.Description.String
			if len(desc) > 200 {
				desc = desc[:200] + "..."
			}
			fmt.Printf("Description: %s\n", desc)
		}

		if post.PublishedAt.Valid {
			fmt.Printf("Published: %s\n", post.PublishedAt.Time.Format("2006-01-02 15:04:05"))
		}

		fmt.Println(strings.Repeat("-", 80))
	}

	return nil
}

