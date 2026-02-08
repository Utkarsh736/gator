
# Gator - RSS Feed Aggregator CLI

A command-line RSS feed aggregator built in Go. Gator allows you to follow RSS feeds, aggregate posts, and browse them directly in your terminal.

## Features

- 👤 User management with authentication
- 📰 Follow multiple RSS feeds
- 🔄 Automatic feed aggregation in the background
- 📖 Browse posts from followed feeds
- 🗄️ PostgreSQL database for persistent storage
- ⚡ Type-safe database queries with SQLC

## Prerequisites

Before running Gator, ensure you have the following installed:

- **Go 1.21 or later** - [Download Go](https://golang.org/dl/)
- **PostgreSQL 15 or later** - [Download PostgreSQL](https://www.postgresql.org/download/)

## Installation

### 1. Install Gator

```bash
go install github.com/Utkarsh736/gator@latest
```

This will install the `gator` binary to your `$GOPATH/bin` directory. Make sure this directory is in your PATH.

### 2. Set Up PostgreSQL

Start PostgreSQL and create a database:

```bash
# Start PostgreSQL (varies by OS)
# macOS: brew services start postgresql@15
# Linux: sudo service postgresql start

# Connect to PostgreSQL
psql postgres  # or: sudo -u postgres psql

# Create database
CREATE DATABASE gator;
```

### 3. Run Database Migrations

Clone the repository to run migrations:

```bash
git clone https://github.com/Utkarsh736/gator.git
cd gator

# Install Goose for migrations
go install github.com/pressly/goose/v3/cmd/goose@latest

# Run migrations
cd sql/schema
goose postgres "postgres://postgres:yourpassword@localhost:5432/gator?sslmode=disable" up
cd ../..
```

### 4. Configure Gator

Create a configuration file at `~/.gatorconfig.json`:

```json
{
  "db_url": "postgres://postgres:yourpassword@localhost:5432/gator?sslmode=disable"
}
```

Replace `yourpassword` with your PostgreSQL password.

## Usage

### User Management

**Register a new user:**
```bash
gator register <username>
```

**Login as a user:**
```bash
gator login <username>
```

**List all users:**
```bash
gator users
```

### Feed Management

**Add a new feed:**
```bash
gator addfeed "<feed_name>" "<feed_url>"
```

Example:
```bash
gator addfeed "Boot.dev Blog" "https://blog.boot.dev/index.xml"
```

**List all feeds:**
```bash
gator feeds
```

**Follow an existing feed:**
```bash
gator follow "<feed_url>"
```

**Unfollow a feed:**
```bash
gator unfollow "<feed_url>"
```

**List feeds you're following:**
```bash
gator following
```

### Aggregation

**Start the feed aggregator:**
```bash
gator agg <duration>
```

This runs continuously and fetches feeds at the specified interval. Examples:
```bash
gator agg 1m    # Fetch every 1 minute
gator agg 30s   # Fetch every 30 seconds
gator agg 1h    # Fetch every 1 hour
```

Press `Ctrl+C` to stop the aggregator.

**Pro tip:** Run the aggregator in a separate terminal window and leave it running in the background!

### Browse Posts

**Browse recent posts from followed feeds:**
```bash
gator browse [limit]
```

Examples:
```bash
gator browse      # Show 2 most recent posts (default)
gator browse 5    # Show 5 most recent posts
gator browse 10   # Show 10 most recent posts
```

### Utility Commands

**Reset database (delete all users and data):**
```bash
gator reset
```

## Example Workflow

```bash
# 1. Register and login
gator register alice
gator login alice

# 2. Add some feeds
gator addfeed "TechCrunch" "https://techcrunch.com/feed/"
gator addfeed "Hacker News" "https://news.ycombinator.com/rss"
gator addfeed "Boot.dev Blog" "https://blog.boot.dev/index.xml"

# 3. Start aggregator (in a separate terminal)
gator agg 1m

# 4. Browse posts
gator browse 5
```

## RSS Feed Suggestions

Here are some popular RSS feeds to get you started:

- **TechCrunch:** `https://techcrunch.com/feed/`
- **Hacker News:** `https://news.ycombinator.com/rss`
- **Boot.dev Blog:** `https://blog.boot.dev/index.xml`
- **The Changelog:** `https://changelog.com/podcast/feed`
- **Lane's Blog:** `https://www.wagslane.dev/index.xml`

## Architecture

Gator is built with:

- **Go** - Core language
- **PostgreSQL** - Database
- **SQLC** - Type-safe SQL query generation
- **Goose** - Database migration management
- **Standard library** - HTTP client, XML parsing, time management

## Development

### Project Structure

```
gator/
├── main.go                  # Entry point
├── commands.go              # Command handlers
├── rss.go                   # RSS feed fetching and parsing
├── internal/
│   ├── config/             # Configuration management
│   │   └── config.go
│   └── database/           # Generated SQLC code
├── sql/
│   ├── schema/             # Goose migrations
│   └── queries/            # SQLC queries
└── sqlc.yaml               # SQLC configuration
```

### Running in Development

```bash
# Run directly with Go
go run . <command> [args...]

# Example
go run . register alice
go run . browse 5
```

### Building from Source

```bash
# Clone the repository
git clone https://github.com/Utkarsh736/gator.git
cd gator

# Build binary
go build -o gator

# Run the binary
./gator register alice
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - feel free to use this project however you'd like!

## Author

**Utkarsh** - [GitHub](https://github.com/Utkarsh736)

---

Built as part of the [Boot.dev](https://boot.dev) backend development curriculum.


