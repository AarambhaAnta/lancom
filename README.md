# lancom

LAN-based TCP communication tool written in Go.

## Current features

- TCP server-client communication
- Line-based message framing

## How to run

**Terminal 1:**

```bash
go run server.go
```

**Terminal 2:**

```bash
go run client.go
```

## Intended features

A full **LAN-based encrypted communication system** with:

- Peer discovery
- Secure key exchange
- Encrypted messaging
- Protocol design
- CLI tool
- Clean architecture

## Project Roadmap

### Stage 0 - Foundation

**Task:**

Build a **simple TCP chat between two terminals**.

- One program acts as server
- Another program connects as client
- They exchange messages continuously
- use goroutines to handle read/ write concurrently

### stage 1 - LAN Peer Discovery

This is where my "real system" begins.

**Task:**

Implement a UDP **broadcast-based peer discovery service**.

- On startup: send broadcast "I am alive: {ip, port}"
- Every peer maintains a map: `{peerID -> (ip, lastSeen)}`
- Remove peers if not seen for X seconds
- Print live peer list every 5 seconds

### stage 2 - Secure Key Exchange

Here I have to learn crypto knowledge and use Go's safe primitives:

- `crypto/elliptic`
- `crypto/hkdf`
- `crypto/ciper/aes`
- `cipher.NewGCM`

**Task:**

Implement a **basic ECDH handshake:**

1. Exchange public keys
2. Compute shared key
3. Derive symmetric key using HKDF
4. Use AES-GCM for encryption/decryption

### stage 3 - Build the Encrypted Channel

Now combine Stage 1( discovery ) + Stage 2( security ) + Stage 0( communication )

**Task:**

- When two peers discover each other -> attempt secure handshake
- Once handshake is done -> establish encrypted TCP channel
- Send all messages encrypted
- Add message framing ( length prefix )
- Handle dropped connections

### stage 4 - Design a Mini Internal Protocol

A real communication system needs structure, not random strings. So, A custom protocol for framing data is good for system.

**Define a protocol:**

Example:

```bash
MSG_TYPE | BODY_LENGTH | BODY
```

Example messages:

- PING
- PONG
- TEXT
- FILE_START
- FILE_CHUNK
- FILE_END
- SESSION_CLOSE

We need to **serialize with JSON or msgpack**

### stage 5 - Final CLI TOOL

Finally wrap everything into a clean, usable **CLI binary:**

Samples:

```bash
lancom list
lancom chat <peerID>
lancom send-file <peerID> <file>
```

We could use **Cobra or simple flags**

Sample package structure:

```bash
/cmd
/core/crypto
/core/discovery
/core/protocol
/core/transport
```
