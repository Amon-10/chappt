# Chappt

Chappt is a small, concurrent terminal chat application written in Go. A TCP
server owns the active-user list and broadcasts newline-delimited messages to
connected terminal clients.

## Features

- Multiple concurrent clients with join and leave notifications
- Case-insensitive duplicate-username protection
- Username retry without reconnecting
- Optional rejection of empty and whitespace-only messages
- Color-aware terminal UI with a clean non-color fallback
- Configurable server and client addresses
- Race-tested three-client joins, messages, disconnects, and reconnects

## Requirements

- Go 1.26 or newer, as declared in `go.mod`

## Run the application

Start the server from the repository root:

```sh
go run ./cmd/server
```

Then open another terminal for each client:

```sh
go run ./cmd/client
```

The default server listens on `:8080`, and the client connects to
`localhost:8080`. Use `Ctrl+D` to leave a client cleanly.

## Run the server with Docker

Build the server image from the repository root:

```sh
docker build -t chappt-server .
```

Run the container and publish its TCP port:

```sh
docker run --rm --name chappt-server -p 8080:8080 chappt-server
```

The terminal client continues to run on the host:

```sh
go run ./cmd/client -addr localhost:8080
```

The image uses a multi-stage build. The final image contains only the statically
linked server binary, runs as a non-root numeric user, and listens on port 8080
by default. Server flags can be appended to the `docker run` command. For
example, `chappt-server -reject-empty=false` allows empty messages.

### Configuration

Server flags:

```text
-addr string
      TCP address to listen on (default ":8080")
-reject-empty
      reject blank or whitespace-only messages (default true)
```

For example, to allow empty messages on port 9000:

```sh
go run ./cmd/server -addr :9000 -reject-empty=false
```

Client flags:

```text
-addr string
      chat server address (default "localhost:8080")
-no-color
      disable ANSI colors
```

For example:

```sh
go run ./cmd/client -addr localhost:9000
```

Color is automatically disabled when standard output is redirected or when the
[`NO_COLOR`](https://no-color.org/) environment variable is set.

## Username and message behavior

Usernames are trimmed before registration. Empty usernames are rejected, and
active usernames must be unique without regard to letter case. For example,
`Alice` and `alice` cannot be connected at the same time. A rejected client is
prompted for another name on the existing connection. Once a user disconnects,
their name becomes available again.

The server controls empty-message behavior. With the default
`-reject-empty=true`, a sender receives a private system notice and the blank
message is not broadcast. Set the flag to `false` to broadcast empty or
whitespace-only messages.

## Wire protocol

Chappt uses UTF-8 text frames separated by newlines:

1. A client sends a username.
2. The server replies with `OK welcome to Chappt` or an `ERR ` line containing
   the rejection reason.
3. After an `OK` response, each client line is treated as a chat message.
4. Chat broadcasts use `username: message`; system events begin with `# `.

The connection remains in the registration phase after an `ERR`, allowing the
client to submit another username.

## Architecture

- `cmd/server` accepts connections and routes join, leave, and message events
  through one manager goroutine. That goroutine exclusively owns the client and
  username maps, making duplicate checks and name reservations atomic.
- `cmd/client` handles registration synchronously, then receives broadcasts in
  a goroutine while the main goroutine reads terminal input.
- `internal/protocol` contains response and system-message markers shared by
  both commands.

## Tests

Run all tests:

```sh
go test ./...
```

Run the suite with the race detector:

```sh
go test -race ./...
```

The integration suite starts the server on an available local TCP port. It
checks server-side empty-name validation, case-insensitive duplicate handling,
retrying on the same connection, both empty-message modes, and a repeated
three-client scenario with messages and reconnect cycles.
