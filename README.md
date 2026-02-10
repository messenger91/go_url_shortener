# URL Shortener Service

A lightweight and efficient URL shortening service built with Go. This service allows you to create short aliases for long URLs and redirect users when they visit the short link.

## Features

- **URL Shortening**: Convert long URLs to short, memorable aliases
- **Custom Aliases**: Optionally specify custom aliases for your URLs
- **SQLite Storage**: Persistent storage using SQLite database
- **Basic Authentication**: Secure endpoint for creating new short URLs
- **RESTful API**: Clean and intuitive API design
- **Logging**: Comprehensive logging with different levels (local, dev, prod)
- **Configuration**: Flexible configuration via YAML files and environment variables

## Prerequisites

- Go 1.25.3 or higher
- SQLite (for storage)

## Installation and Setup

### 1. Clone the repository

```bash
git clone https://github.com/messenger91/go_url_shortener.git
cd go_url_shortener
```

### 2. Install dependencies

```bash
go mod tidy
```

### 3. Configure the application

Create a configuration file or use the provided `config/local.yaml` as a template. The configuration supports the following options:

- `version`: Application version
- `env`: Environment (local, dev, prod)
- `storage_path`: Path to the SQLite database file
- `alias_length`: Default length for auto-generated aliases
- `http_server`: Server configuration (address, timeouts, credentials)

### 4. Set environment variables

```bash
export CONFIG_PATH=./config/local.yaml
export HTTP_SERVER_PASSWORD=admin  # Or set in config file
```

### 5. Run the application

```bash
go run cmd/url-shortener/main.go
```

The server will start on the configured address (default: `localhost:8082`).

## Configuration

The application uses a YAML configuration file that can be overridden by environment variables. Here's the default configuration:

```yaml
version: "0.1.0"
env: "local"                    # local, dev, prod
storage_path: "./storage.db"   # SQLite DB file path
alias_length: 8                # Default length for generated aliases
http_server:                   # HTTP server configuration
  address: "localhost:8082"    # Server address
  timeout: 4s                  # Request timeout
  idle_timeout: 30s            # Idle connection timeout
  user: "admin"               # Basic auth username
  password: "admin"           # Basic auth password
```

Environment variables that can override configuration values:
- `CONFIG_PATH` - Path to the configuration file
- `HTTP_SERVER_PASSWORD` - Password for basic authentication

## API Endpoints

### Create a Short URL

**POST** `/url`

Creates a new short URL alias for the provided long URL.

#### Headers
- `Content-Type: application/json`
- `Authorization: Basic <credentials>` (Base64 encoded username:password)

#### Request Body
```json
{
  "url": "https://www.example.com/very/long/url/to/be/shortened",
  "alias": "optional_custom_alias"
}
```

- `url` (required): The long URL to be shortened (must be a valid URL)
- `alias` (optional): Custom alias for the short URL. If not provided, a random alias of configured length will be generated.

#### Example Request
```bash
curl -X POST http://localhost:8082/url \
  -H "Content-Type: application/json" \
  -H "Authorization: Basic $(echo -n 'admin:admin' | base64)" \
  -d '{"url": "https://www.example.com/very/long/url/to/be/shortened", "alias": "mylink"}'
```

#### Response
Success (200 OK):
```json
{
  "operation": "OK",
  "alias": "mylink"
}
```

If no alias was provided in the request:
```json
{
  "operation": "OK",
  "alias": "a1b2c3d4"
}
```

Error responses:
- 400: Invalid URL format
- 409: URL already exists
- 500: Internal server error

### Redirect to Original URL

**GET** `/{alias}`

Redirects to the original URL associated with the given alias.

#### Path Parameters
- `alias` (required): The short alias to look up

#### Example Request
```bash
curl -L http://localhost:8082/mylink
```

This will return a 302 redirect to the original URL.

#### Response
- 302 Found: Redirect to the original URL
- 404 Not Found: Alias not found
- 500 Internal Server Error: Database error

## Development

### Running Tests

```bash
go test ./...
```

### Project Structure

```
.
├── cmd/
│   └── url-shortener/
│       └── main.go             # Application entry point
├── config/
│   └── local.yaml             # Local configuration
├── internal/
│   ├── config/                # Configuration management
│   ├── http-server/           # HTTP server implementation
│   │   ├── handlers/          # Request handlers
│   │   │   ├── url/
│   │   │   │   └── save/      # Save URL handler
│   │   │   └── redirect/      # Redirect handler
│   │   └── middleware/        # HTTP middleware
│   ├── lib/                   # Utility libraries
│   └── storage/               # Storage implementations
│       └── sqlite/            # SQLite storage
├── tests/                     # Integration tests
├── go.mod                     # Go module definition
├── go.sum                     # Go module checksums
└── README.md                  # This file
```

## Environment Variables

- `CONFIG_PATH`: Path to the configuration file (required)
- `HTTP_SERVER_PASSWORD`: Override the password in the config file

## Security

- Basic authentication is required for creating new short URLs
- Input validation is performed on all incoming URLs
- SQL injection is prevented through prepared statements

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.