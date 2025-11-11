# Minesweeper for 3270 Terminals

Copyright 2025 by moshix. All rights reserved. 

A multi-user minesweeper game for IBM 3270 terminals, built with the `go3270` library. This project demonstrates key concepts of 3270 terminal programming including connection handling, session management, terminal size detection, and keyboard input handling.

## Quick Start

### Building and Running

```bash
go build -o minesweeper
./minesweeper
```

### Conecting

```bash
x3270 localhost:3270    # GUI emulator
c3270 localhost:3270    # Terminal-based emulator
```

### Game Controls

- **ENTER**: Reveal cell at cursor
- **F1**: Toggle flag at cursor
- **F3**: New game
- **F12**: Quit

## Project Structure

```
.
├── config.go          # Configuration file parser
├── game.go            # Game logic (mines, reveals, win/loss)
├── screen.go          # Screen building (rendering game state)
├── minesweeper.go     # Main server (connections, event loop)
├── mine.cnf           # Configuration file
├── go.mod             # Go module definition
└── README.md          # This file
```

## Configuration

Edit `mine.cnf`:

```ini
# Minesweeper configuration
port=3270
instance_name=Minesweeper Server
```

- `port`: TCP port (default: 3270)
- `instance_name`: Server name shown to players

## Features

- **Multi-user**: Each connection is isolated, no shared state
- **Adaptive board**: Automatically sizes to terminal dimensions
- **First-click protection**: First revealed cell is never a mine
- **Flood fill**: Empty cells auto-reveal neighbors
- **Color-coded**: Uses 3270 color attributes (blue, green, red, yellow, turquoise)
---

## Understanding the Implementation

This guide explains the core concpts for developers new to the `go3270` library.

### 1. Connection Handling

The `go3270` library uses standard Go networking but adds 3270-specific protocol negotiation. Here's how connections work:

```go
// Accept TCP connections in a loop
for {
    conn, err := listener.Accept()
    if err != nil {
        log.Printf("Accept error: %v", err)
        continue
    }

    // Handle each connection in a separate goroutine
    go handleConnection(conn, config)
}
```

Each connection requires two steps:

**Step 1: Telnet Protocol Negotiation**

The 3270 protocol uses telnet as its transport layer. Before you can send screens, you must negotiate the connection:

```go
// Set timeout for negotiation phase
conn.SetDeadline(time.Now().Add(30 * time.Second))

// Negotiate telnet protocol and get termnal information
devInfo, err := go3270.NegotiateTelnet(conn)
if err != nil {
    log.Printf("Telnet negotiation failed: %v", err)
    return
}

// Clear deadline after succesful negotiation
conn.SetDeadline(time.Time{})
```

`NegotiateTelnet()` returns a `devInfo` object containing terminal capabilities, including screen dimensions.

**Step 2: Screen Interaction**

After negotiation, use `go3270.HandleScreen()` to display screens and wait for user input:

```go
resp, err := go3270.HandleScreen(
    screen,                              // The screen to display
    nil,                                 // Optional input fields (not used here)
    nil,                                 // Optional field map
    []go3270.AID{go3270.AIDEnter},       // "Accept" keys (continue processing)
    []go3270.AID{go3270.AIDPF12},        // "Reject" keys (exit/cancel)
    "",                                  // Input field name
    termRows/2,                          // Initial cursor row
    termCols/2,                          // Initial cursor column
    conn,                                // Network connection
)
```

The function returns a `resp` object containing:
- Which key (AID) teh  user pressed
- Cursor position when the key was pressed
- Any field data if using input fields

---

### 2. User Session Structure (Multitenancy)

Each player connection runs in its own goroutine with isolated state. This eliminates teh need for a database while supporting multiple simultaneous players.

**Session Isolation**

```go
func handleConnection(conn net.Conn, config *Config) {
    defer conn.Close()

    remoteAddr := conn.RemoteAddr().String()
    log.Printf("New connection from %s", remoteAddr)
    
    // ... negotiation code ...
    
    // Start THIS player's game loop
    playGame(conn, config, termRows, termCols, remoteAddr)
}
```

**Per-Player Game State**

Each player gets their own `Game` instance:

```go
func playGame(conn net.Conn, config *Config, termRows, termCols int, remoteAddr string) {
    // Calculate board size for THIS player's terminal
    boardRows := (maxBoardHeight - 1) / 2
    boardCols := (maxBoardWidth - 1) / 4
    
    // Create a NEW game instance for this player
    game := NewGame(boardRows, boardCols)
    
    // Main game loop - isolated to this connection
    for {
        screen := BuildGameScreen(game, config, termRows, termCols)
        resp, err := go3270.HandleScreen(/* ... */)
        
        // Process input and update THIS game instance
        switch resp.AID {
        case go3270.AIDEnter:
            game.RevealCell(boardRow, boardCol)
        // ...
        }
    }
}
```

**The Game Strutcure**

Each player's game state is completely independent:

```go
type Game struct {
    Board         [][]Cell     // This player's board
    Rows          int          // Board dimensions
    Cols          int
    TotalMines    int          // Mine count
    FlagsPlaced   int
    CellsRevealed int
    Status        GameStatus   // Playing/Won/Lost
    FirstMove     bool         // First click protection
    rng           *rand.Rand   // Random number generator
}
```

When the player disconnects, the goroutine ends and Go's garbage collector auomatically cleans up the `Game` instance.

---

### 3. Terminal Size Detection and Adaptation

3270 terminals come in various sizes (24x80, 32x80, 43x80, 62x160, etc.). Your application should adapt to whatever size the user's terminal provides.

**Getting Terminal Dimensions**

```go
// After negotiation, extract terminal size
termRows, termCols := devInfo.AltDimensions()
```

**Adapting the Game Board**

The minesweeper board uses a box-drawing layout where each cell occupies:
- **4 columns**: `|XXX` (border + 3 content characters)
- **2 rows**: one for borders (`+---+`), one for content (`| X |`)

Calculate the maximum board size that fits:

```go
// Reserve space for UI elements (title, status, instructions)
// Reserve: title(1) + blank(1) + status(1) + blank(1) + boardStart(1) + legend(3) = 7 rows
maxBoardHeight := termRows - 7
boardRows := (maxBoardHeight - 1) / 2  // Each cell is 2 rows tall

maxBoardWidth := termCols - 2          // Leave 1 column margin on each side
boardCols := (maxBoardWidth - 1) / 4   // Each cell is 4 columns wide

// Apply constraints
if boardRows < 5 {
    boardRows = 5  // Minimum playable size
}
if boardCols > 15 {
    boardCols = 15  // Maximum for reasonable gameplay
}
```

**Dynamic Screen Layout**

Center the board on the screen:

```go
// Calculate actual board width in characters
boardWidth := 1 + game.Cols*4

// Center horizontally
boardStartCol := (termCols - boardWidth) / 2
if boardStartCol < 0 {
    boardStartCol = 0
}

// Fixed vertical position
boardStartRow := 5
```

**Example Size Adaptations**

| Terminal Size | Board Size | Total Cells | Mines |
|--------------|------------|-------------|-------|
| 24x80 (Model 2) | 8x10 | 80 | 13 |
| 32x80 (Model 3) | 11x10 | 110 | 18 |
| 43x80 (Model 4) | 18x10 | 180 | 30 |
| 62x160 (Model 5) | 27x39 | 1,053 | 175 |

---

### 4. AID Key Handling

AID (Attention IDentifier) keys is how 3270 terminals send commands. Unlike regular terminals that stream characters, 3270 terminals are "screen-at-a-time" devices.

**Understanding AID Keys**

When a user presses an AID key:
1. The entire screen buffer (and cursor position) is sent to the host
2. Processing waits until `HandleScreen()` returns
3. Your code handles the key and sends a new screen

**Configuring Accepted Keys**

`HandleScreen()` takes two lists of AID keys:

```go
resp, err := go3270.HandleScreen(
    screen,
    nil,
    nil,
    []go3270.AID{go3270.AIDEnter},              // "Accept" keys
    []go3270.AID{go3270.AIDPF1, go3270.AIDPF3, go3270.AIDPF12},  // "Reject" keys
    // ...
)
```

- **Accept keys**: Processing continues normally
- **Reject keys**: Often used for exit/cancel actions, but your code still processes them

**Processing AID Keys**

Check which key was pressed and the cursor position:

```go
switch resp.AID {
case go3270.AIDEnter:
    // Get cursor position
    cursorRow := resp.Row
    cursorCol := resp.Col
    
    // Convert screen coordinates to game board coordinates
    boardRow := (cursorRow - boardStartRow - 1) / 2
    boardCol := (cursorCol - boardStartCol - 1) / 4
    
    // Validate and perform action
    if boardRow >= 0 && boardRow < game.Rows && 
       boardCol >= 0 && boardCol < game.Cols {
        game.RevealCell(boardRow, boardCol)
    }

case go3270.AIDPF1:
    // Flag/unflag cell (same coordinate conversion)
    boardRow := (cursorRow - boardStartRow - 1) / 2
    boardCol := (cursorCol - boardStartCol - 1) / 4
    game.ToggleFlag(boardRow, boardCol)

case go3270.AIDPF3:
    // New game - create fresh game instance
    game = NewGame(boardRows, boardCols)

case go3270.AIDPF12:
    // Quit - exit the game loop
    return

default:
    // Unknown/unhandled key - ignore
}
```

**Common AID Keys**

| Constant | Key | Typical Use |
|----------|-----|-------------|
| `AIDEnter` | ENTER | Submit/select |
| `AIDPF1` - `AIDPF24` | F1-F24 | Function keys |
| `AIDPA1` - `AIDPA3` | PA1-PA3 | Program attention |
| `AIDClear` | CLEAR | Clear screen |

**Coordinate Conversion**

You can read the cursor position at the time the user pressed an AID key. This is helpful so the user can point at a row or column (or row and column) and you can then take action depending what the user is pointing the cursor at. In our game the user will put the cursor at the cell they want to uncover. 

Since the game board doesn't start at (0,0) and cells span multple characters, you must convert screen coordinates to board coordinates:

```go
// Board starts at (boardStartRow, boardStartCol)
// First cell content is at (boardStartRow + 1, boardStartCol + 1)
// Cells are spaced 2 rows and 4 columns apart

// Subtract starting position and borders
boardRow := (cursorRow - boardStartRow - 1) / 2
boardCol := (cursorCol - boardStartCol - 1) / 4

// The cursor can be anywhere within the cell's 3-character content area,
// so division automatically gives the correct cell index
```

---



## Dependencys

- `github.com/racingmars/go3270` - 3270 terminal protocol library

## License

Demonstration project for 3270 terminal applications.

## Resources

- [go3270 documentation](https://pkg.go.dev/github.com/racingmars/go3270)
- [3270 Data Stream Programmer's Reference](https://publibfp.dhe.ibm.com/epubs/pdf/ga23502c.pdf)
- [x3270 emulator](http://x3270.bgp.nu/)
