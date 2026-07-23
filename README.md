# lancom

A terminal chat room over TCP on the local network, written in Go. One server, many clients; each client gets a scrollable message pane with an input box pinned to the bottom of the terminal.

## Features

- Broadcast messages to everyone in the room
- Private messages to one or several chosen recipients (`/msg`)
- Nicknames (`/nick`)
- Online roster (`/list`)
- Graceful leave, announced to the room whether you `/quit` or just close the terminal

## How to run

**Terminal 1 — start the server:**

```bash
go run ./cmd/server
```

**Terminal 2+ — one per person joining the room:**

```bash
go run ./cmd/client
```

## Commands

| Command | Effect |
| --- | --- |
| `<text>` | Broadcast `<text>` to everyone in the room |
| `/msg <nick[,nick...]> <text>` | Private message one or more recipients by nickname, comma-separated |
| `/nick <name>` | Change your nickname (must be unique, 3+ characters, not reserved) |
| `/list` | Show who's currently online |
| `/quit` | Leave the room |

## Architecture

```text
cmd/server   TCP accept loop, one goroutine per connected client, routes
             messages by type, owns the nickname/roster state
cmd/client   Bubble Tea TUI — scrolling message pane + input box fixed
             to the bottom of the terminal
protocol     Wire format shared by both: a JSON message framed by a
             trailing newline
```

**Wire format.** Each message is a single line of JSON — `{"version","type","from","to","body"}` — terminated by `\n`. Line framing keeps the transport trivial (`bufio.Reader.ReadString('\n')` on both ends) while JSON keeps every message human-readable on the wire, which matters more than a compact binary format at this scale.

**Message types**, all client↔server, following a `*_req`/`*_ack` pattern:

| Type | Direction | Purpose |
| --- | --- | --- |---|
| `join_req` / `join_ack` | → / ← | Handshake; server assigns a client ID |
| `chat` | → and ← | Broadcast to the room |
| `chat_ack` | ← | Delivery confirmation to the sender only |
| `dm` | → and ← | Private message; `to` holds one or more comma-separated nicknames |
| `dm_ack` | ← | Delivery status (delivered count / not-found nicknames) |
| `nick_req` / `nick_ack` | → / ← | Nickname change request / confirmation |
| `list_req` / `list_ack` | → / ← | Roster request / comma-separated online nicknames |
| `leave` | → | Explicit "I'm leaving" — also inferred if the connection just drops |
| `error_message` | ← | Any handler error, sent back to whoever caused it |

**Why the server never parses chat text.** Commands (`/msg`, `/nick`, `/list`) are parsed client-side into their own typed messages before they're sent — the server only ever switches on `Type`. An earlier version sniffed `/` and `@` prefixes out of chat body text on the server; routing by type instead means the server has one job (route a typed message to the right handler) and the wire protocol stays the single source of truth for what a client can ask for.

**Concurrency.** Each client connection runs in its own goroutine (`clientHandler`); a single mutex guards the shared `clients`/`nickNames` maps. Broadcasts and DMs snapshot the recipient list under the lock, then release it before doing any network I/O, so a slow client can't stall the room.

## Possible future work

Not active scope, but natural next steps if this grows further: LAN peer discovery (no hardcoded server address), end-to-end encryption, and a persistent chat history.
