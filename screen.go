package main
/* minesweeper for 3270 terminals
   copyright 2025 by moshix
   all rights reserved
*/


import (
	"fmt"
	"strings"

	"github.com/racingmars/go3270"
)

// BuildGameScreen creates the screen for the minesweeper game with box borders in green
func BuildGameScreen(game *Game, config *Config, termRows, termCols int) go3270.Screen {
	screen := go3270.Screen{}

	// Calculate board dimensions with boxes
	// Each cell is 4 chars wide (|XXX), plus 1 for final border = 1 + cols*4
	boardWidth := 1 + game.Cols*4

	// Calculate board offset to center it
	boardStartRow := 5
	boardStartCol := (termCols - boardWidth) / 2
	if boardStartCol < 0 {
		boardStartCol = 0
	}

	// Title (Blue)
	titleText := config.InstanceName
	titleCol := (termCols - len(titleText)) / 2
	screen = append(screen, go3270.Field{
		Row:     0,
		Col:     titleCol,
		Content: titleText,
		Color:   go3270.Blue,
		Intense: true,
	})

	// Terminal size in top right crner 
	termSizeText := fmt.Sprintf("%dx%d", termRows, termCols)
	termSizeCol := termCols - len(termSizeText) - 10
	screen = append(screen, go3270.Field{
		Row:     0,
		Col:     termSizeCol,
		Content: termSizeText,
		Color:   go3270.White,
	})

	// Status line (Green)
	var statusText string
	switch game.Status {
	case StatusPlaying:
		minesLeft := game.TotalMines - game.FlagsPlaced
		statusText = fmt.Sprintf("Mines: %d   Flags: %d", minesLeft, game.FlagsPlaced)
	case StatusWon:
		statusText = "*** YOU WIN! ***"
	case StatusLost:
		statusText = "*** GAME OVER ***"
	}

	statusCol := (termCols - len(statusText)) / 2
	screen = append(screen, go3270.Field{
		Row:     2,
		Col:     statusCol,
		Content: statusText,
		Color:   go3270.Green,
		Intense: true,
	})

	// Draw the game board with boxes
	for row := 0; row <= game.Rows; row++ {
		// Draw horizontal border line
		screenRow := boardStartRow + row*2
		var borderLine strings.Builder
		borderLine.WriteString("+")
		for col := 0; col < game.Cols; col++ {
			borderLine.WriteString("---+")
		}

		screen = append(screen, go3270.Field{
			Row:     screenRow,
			Col:     boardStartCol,
			Content: borderLine.String(),
			Color:   go3270.Green,
		})

		// Draw cell content line (if not the last border)
		if row < game.Rows {
			screenRow = boardStartRow + row*2 + 1

			// Build the complete line with all content
			var lineBuilder strings.Builder
			lineBuilder.WriteString("|")

			for col := 0; col < game.Cols; col++ {
				cell := &game.Board[row][col]

				var cellText string
				switch cell.State {
				case StateHidden:
					cellText = " . "
				case StateFlagged:
					cellText = " F "
				case StateRevealed:
					if cell.IsMine {
						cellText = " * "
					} else if cell.AdjacentMines > 0 {
						cellText = fmt.Sprintf(" %d ", cell.AdjacentMines)
					} else {
						cellText = "   "
					}
				}

				lineBuilder.WriteString(cellText)
				lineBuilder.WriteString("|")
			}

			// Draw the complete line in green (for structure)
			screen = append(screen, go3270.Field{
				Row:     screenRow,
				Col:     boardStartCol,
				Content: lineBuilder.String(),
				Color:   go3270.Green,
			})

			// Now overlay colored content for flags and mines
			for col := 0; col < game.Cols; col++ {
				cell := &game.Board[row][col]
				cellCol := boardStartCol + 1 + col*4

				if cell.State == StateFlagged {
					screen = append(screen, go3270.Field{
						Row:     screenRow,
						Col:     cellCol,
						Content: " F ",
						Color:   go3270.Red,
						Intense: true,
					})
				} else if cell.State == StateRevealed && cell.IsMine {
					screen = append(screen, go3270.Field{
						Row:     screenRow,
						Col:     cellCol,
						Content: " * ",
						Color:   go3270.Yellow,
						Intense: true,
					})
				}
			}
		}
	}

	// Instructons 
	instructions := "ENTER=Reveal  F1=Flag  F3=New Game  F12=Quit"
	instrCol := (termCols - len(instructions)) / 2
	// Place 2 rows below the last border line of the board
	instrRow := boardStartRow + game.Rows*2 + 2
		instrRow = 3
		instrCol = 19
	screen = append(screen, go3270.Field{
		Row:     instrRow,
		Col:     instrCol,
		Content: instructions,
		Color:   go3270.Turquoise,
		Intense: true,
	})

	return screen
}

// BuildWelcomeScreen creates an initial welcome screen
func BuildWelcomeScreen(config *Config, termRows, termCols int) go3270.Screen {
	screen := go3270.Screen{}

	// Title
	titleText := config.InstanceName
	titleCol := (termCols - len(titleText)) / 2
	screen = append(screen, go3270.Field{
		Row:     termRows/2 - 3,
		Col:     titleCol,
		Content: titleText,
		Color:   go3270.Blue,
		Intense: true,
	})

	// Welcome message
	welcomeText := "Welcome to Minesweeper for 3270!"
	welcomeCol := (termCols - len(welcomeText)) / 2
	screen = append(screen, go3270.Field{
		Row:     termRows/2 - 1,
		Col:     welcomeCol,
		Content: welcomeText,
		Color:   go3270.White,
	})

	// Instructions
	instructions := []string{
		"Press ENTER to start a new game",
		"",
		"Game Controls:",
		"  ENTER = Reveal cell at cursor",
		"  F1    = Toggle flag at cursor",
		"  F3    = New game",
		"  F12   = Quit",
	}

	startRow := termRows/2 + 1
	for i, line := range instructions {
		lineCol := (termCols - len(line)) / 2
		color := go3270.Turquoise
		if i >= 2 {
			color = go3270.Green
		}
		screen = append(screen, go3270.Field{
			Row:     startRow + i,
			Col:     lineCol,
			Content: line,
			Color:   color,
		})
	}

	return screen
}
