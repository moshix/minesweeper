package main
/* minesweeper for 3270 terminals
   copyright 2025 by moshix
   all rights reserved
*/
import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/racingmars/go3270"
)

const (
	defaultConfigFile = "mine.cnf"
)

func main() {
	// Load configuration
	config, err := LoadConfig(defaultConfigFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Starting %s on port %d", config.InstanceName, config.Port)

	// Start TCP server
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", config.Port))
	if err != nil {
		log.Fatalf("Failed to start listener: %v", err)
	}
	defer listener.Close()

	log.Printf("Server listening on port %d", config.Port)

	// Accept connections in a loop
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}

		// Handle each connection in a separate goroutine (multi-user support)
		go handleConnection(conn, config)
	}
}

// handleConnection handles each singl player connection
func handleConnection(conn net.Conn, config *Config) {
	defer conn.Close()

	remoteAddr := conn.RemoteAddr().String()
	log.Printf("New connection from %s", remoteAddr)

	// Set initial timeout for telnet negotiation
	conn.SetDeadline(time.Now().Add(30 * time.Second))

	// Negotiate telnet protocol and get terminal information
	devInfo, err := go3270.NegotiateTelnet(conn)
	if err != nil {
		log.Printf("Telnet negotiation failed for %s: %v", remoteAddr, err)
		return
	}

	// Clear deadline after negotiation
	conn.SetDeadline(time.Time{})

	// Get actual terminal size from device info
	termRows, termCols := devInfo.AltDimensions()

	// Show welcome screen
	welcomeScreen := BuildWelcomeScreen(config, termRows, termCols)
	_, err = go3270.HandleScreen(
		welcomeScreen,
		nil,
		nil,
		[]go3270.AID{go3270.AIDEnter},
		[]go3270.AID{go3270.AIDPF12},
		"",
		termRows/2,
		termCols/2,
		conn,
	)

	if err != nil {
		log.Printf("Error showing welcome screen to %s: %v", remoteAddr, err)
		return
	}

	// Start game loop
	playGame(conn, config, termRows, termCols, remoteAddr)

	log.Printf("Connection closed: %s", remoteAddr)
}

// playGame runs the main game loop
func playGame(conn net.Conn, config *Config, termRows, termCols int, remoteAddr string) {
	// Calculate board size based on terminal size
	// With boxes: each cell is 4 chars wide and 2 rows tall, plus borders
	// Board width = 1 + cols*4, Board height = 1 + rows*2
	// Reserve rows for: title(1) + blank(1) + status(1) + blank(1) + boardStart(1) + legend area(3) = 7
	maxBoardHeight := termRows - 7
	boardRows := (maxBoardHeight - 1) / 2 // Subtract 1 for top border, divide by 2 for cell height
	if boardRows < 5 {
		boardRows = 5
	}

	maxBoardWidth := termCols - 2
	boardCols := (maxBoardWidth - 1) / 4 // Subtract 1 for left border, divide by 4 for cell width
	if boardCols > 15 {
		boardCols = 15 // Cap at reasonable size (boxes take more space)
	}
	if boardCols < 5 {
		boardCols = 5
	}

	// Create new game
	game := NewGame(boardRows, boardCols)

	log.Printf("Started new game for %s: %dx%d board with %d mines",
		remoteAddr, game.Rows, game.Cols, game.TotalMines)

	// Calculate board dimensions and position
	boardWidth := 1 + game.Cols*4
	boardStartRow := 5
	boardStartCol := (termCols - boardWidth) / 2
	if boardStartCol < 0 {
		boardStartCol = 0
	}

	// Calculate initial cursor position (center cell's content area)
	centerRow := game.Rows / 2
	centerCol := game.Cols / 2
	initialCursorRow := boardStartRow + centerRow*2 + 1     // Cell content row
	initialCursorCol := boardStartCol + 2 + centerCol*4 + 1 // Middle of cell content (on the character)

	// this is the main Game loop
	for {
		// Build screen
		screen := BuildGameScreen(game, config, termRows, termCols)

		resp, err := go3270.HandleScreen(
			screen,
			nil,
			nil,
			[]go3270.AID{go3270.AIDEnter},                         // Accept Enter
			[]go3270.AID{go3270.AIDPF1, go3270.AIDPF3, go3270.AIDPF12}, // Accept F1, F3, F12
			"",
			initialCursorRow,
			initialCursorCol,
			conn,
		)

		if err != nil {
			log.Printf("Error handling screen for %s: %v", remoteAddr, err)
			return
		}

		// Handle different AID keys
		switch resp.AID {
		case go3270.AIDEnter:
			// Reveal cell at cursor position
			cursorRow := resp.Row
			cursorCol := resp.Col

			// Convert screen coordinates to board coordinates
			// Cell content is at: boardStartRow + row*2 + 1, boardStartCol + 1 + col*4
			boardRow := (cursorRow - boardStartRow - 1) / 2
			boardCol := (cursorCol - boardStartCol - 1) / 4

			// Validate and reveal
			if boardRow >= 0 && boardRow < game.Rows && boardCol >= 0 && boardCol < game.Cols {
				game.RevealCell(boardRow, boardCol)
			}

		case go3270.AIDPF1:
			// Toggle flag at cursor position
			cursorRow := resp.Row
			cursorCol := resp.Col

			// Convert screen coordinates to board coordinates
			boardRow := (cursorRow - boardStartRow - 1) / 2
			boardCol := (cursorCol - boardStartCol - 1) / 4

			if boardRow >= 0 && boardRow < game.Rows && boardCol >= 0 && boardCol < game.Cols {
				game.ToggleFlag(boardRow, boardCol)
			}

		case go3270.AIDPF3:
			// New game
			game = NewGame(boardRows, boardCols)
			log.Printf("Player %s started new game", remoteAddr)

		case go3270.AIDPF12:
			// Quit
			log.Printf("Player %s quit", remoteAddr)
			return

		default:
			// Unknown key, ignore
		}

		// If game is over, show final screen and wait for new game or quit
		if game.Status != StatusPlaying {
			// Show final screen
			screen = BuildGameScreen(game, config, termRows, termCols)
			resp, err = go3270.HandleScreen(
				screen,
				nil,
				nil,
				nil,
				[]go3270.AID{go3270.AIDPF3, go3270.AIDPF12},
				"",
				initialCursorRow,
				initialCursorCol,
				conn,
			)

			if err != nil {
				log.Printf("Error showing final screen for %s: %v", remoteAddr, err)
				return
			}

			if resp.AID == go3270.AIDPF3 {
				// New game
				game = NewGame(boardRows, boardCols)
				log.Printf("Player %s started new game after game over", remoteAddr)
			} else {
				// Quit or unknown key
				log.Printf("Player %s quit after game over", remoteAddr)
				return
			}
		}
	}
}
