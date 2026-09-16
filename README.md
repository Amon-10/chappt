# Chappt

[![CI](https://github.com/Amon-10/chappt/actions/workflows/ci.yml/badge.svg)](https://github.com/Amon-10/chappt/actions/workflows/ci.yml)

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

### Deployment considerations

Chappt uses long-lived raw TCP connections, so the host must expose a TCP port
rather than an HTTP-only service. Deploy one server instance for this version:
the connected-client and username maps live in memory and are intentionally not
shared between processes. A restart disconnects active clients, which can then
reconnect normally.

After deployment, connect the terminal client to the host and published port:

```sh
go run ./cmd/client -addr chat.example.com:8080
```

The deployment target should provide a TCP health check and keep the container
running while clients are connected. Transport encryption is not included in
V1, so the current image is best suited to a controlled demo environment.

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

```text
terminal input
    │
    ▼
client main goroutine ──TCP line──▶ server connection goroutine
                                      │
                                      ▼
                               broadcast channel
                                      │
                                      ▼
                               manager goroutine
                                      │
                       TCP lines to other connections
                                      │
                                      ▼
                           client receive goroutines
                                      │
                                      ▼
                               terminal output
```

- `cmd/server` accepts connections and starts one handler goroutine for each
  client. Handlers turn network activity into join, leave, and message events.
- The server starts one manager goroutine. It exclusively owns the connection
  and username maps, so duplicate checks and membership changes do not require
  mutexes.
- `cmd/client` registers synchronously. It then reads terminal input in the main
  goroutine and receives server messages in one additional goroutine.
- `internal/protocol` contains the small set of response and system-message
  markers shared by both commands.

### Design decisions

- **Newline framing:** `bufio.Scanner` reads one complete frame at a time, and
  `fmt.Fprintln` terminates every outgoing frame with a newline.
- **One goroutine per connection:** each server handler can block on its own TCP
  read without preventing other clients from sending messages.
- **Channel-owned state:** handlers communicate through `join`, `leave`, and
  `broadcast` channels. Only the manager accesses the maps of active clients
  and normalized usernames.
- **Server-authoritative validation:** the client provides immediate feedback,
  but the server makes the final decision about usernames and empty messages.
- **Small shared protocol package:** shared markers prevent the two executable
  commands from drifting without introducing a larger protocol abstraction.

## Tests

Run all tests:

```sh
go test ./...
```

Run the suite with the race detector:

```sh
go test -race ./...
```

GitHub Actions runs `gofmt`, `go vet`, the normal test suite, and the
race-detector suite on every push and pull request.

The integration suite starts the server on an available local TCP port. It
checks server-side empty-name validation, case-insensitive duplicate handling,
retrying on the same connection, both empty-message modes, and a repeated
three-client scenario with messages and reconnect cycles.
