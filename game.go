package main
/* minesweeper for 3270 terminals
   copyright 2025 by moshix
   all rights reserved
*/


import (
	"math/rand"
	"time"
)
// time needed ?? let's see




// CellState is the state of a cell
type CellState int

const (
	StateHidden CellState = iota
	StateRevealed
	StateFlagged
)

// Cell is a single cell on the minesweeper board
type Cell struct {
	IsMine         bool
	AdjacentMines  int
	State          CellState
}




    // GameStatus the current status of the game
    type GameStatus int
    
    const (
    	StatusPlaying GameStatus = iota
    	StatusWon
    	StatusLost
    )





// Game a minesweeper game instance
type Game struct {
	Board         [][]Cell
	Rows          int
	Cols          int
	TotalMines    int
	FlagsPlaced   int
	   CellsRevealed int
	Status        GameStatus
	StartTime     time.Time
	FirstMove     bool
	rng           *rand.Rand
}

// NewGame creates a new game and adapts to termnal size
// boardRows and boardCols are  the playable area (wthout the header/footer)
func NewGame(boardRows, boardCols int) *Game {
	// Calculate number of mines (15-20% of total cells)
	totalCells := boardRows * boardCols
	mineCount := totalCells / 6 // roughly 16.7%
	if mineCount < 1 {
		mineCount = 1
	}

	game := &Game{
		Rows:       boardRows,
		Cols:       boardCols,
		TotalMines: mineCount,
		Status:     StatusPlaying,
		FirstMove:  true,
		rng:        rand.New(rand.NewSource(time.Now().UnixNano())),
	}

      	// Init board
      	game.Board = make([][]Cell, boardRows)
      	for i := range game.Board {
      		game.Board[i] = make([]Cell, boardCols)
	}

	return game
}

// place mines randomly on the board, avoiding the first clicked cell because of reasons
func (g *Game) PlaceMines(avoidRow, avoidCol int) {
	minesPlaced := 0

	for minesPlaced < g.TotalMines {
		row := g.rng.Intn(g.Rows)
		col := g.rng.Intn(g.Cols)

		// Don't place mine on first click or if already has mine
		if (row == avoidRow && col == avoidCol) || g.Board[row][col].IsMine {
			continue
		}

		g.Board[row][col].IsMine = true
		minesPlaced++
	}

	// Calculate adjacent mine counts
	g.calculateAdjacentMines()
	g.StartTime = time.Now()
}

// calculateAdjacentMines calculates the number of adjacent mines for each cell
func (g *Game) calculateAdjacentMines() {
	for row := 0; row < g.Rows; row++ {
		for col := 0; col < g.Cols; col++ {
			if !g.Board[row][col].IsMine {
				g.Board[row][col].AdjacentMines = g.countAdjacentMines(row, col)
			}
		}
	}
}






      // countAdjacentMines counts mines in the 8 adjacent cells
      func (g *Game) countAdjacentMines(row, col int) int {
      	count := 0
      	for dr := -1; dr <= 1; dr++ {
      		for dc := -1; dc <= 1; dc++ {
      			if dr == 0 && dc == 0 {
      				continue
      			}
      			nr, nc := row+dr, col+dc
      			if nr >= 0 && nr < g.Rows && nc >= 0 && nc < g.Cols {
      				if g.Board[nr][nc].IsMine {
      					count++
      				}
      			}
      		}
      	}
      	return count
      }

// RevealCell reveals a cell at the given position
func (g *Game) RevealCell(row, col int) {
	// Validate coordinates
	if row < 0 || row >= g.Rows || col < 0 || col >= g.Cols {
		return
	}

	cell := &g.Board[row][col]

	// Can't reveal flagged or already revealed cells
	if cell.State != StateHidden {
		return
	}

	// First move: place mines avoiding this cell
	if g.FirstMove {
		g.PlaceMines(row, col)
		g.FirstMove = false
	}

	// Reveal the cell
	cell.State = StateRevealed
	g.CellsRevealed++

	// Hit a mine - game over
	if cell.IsMine {
		g.Status = StatusLost
		g.revealAllMines()
		return
	}

	// If cell has no adjacent mines, reveal neighbors (flood fill)
	if cell.AdjacentMines == 0 {
		g.revealAdjacentCells(row, col)
	}

	// Check for win condition
	g.checkWinCondition()
}

// revealAdjacentCells recursively reveal adjacent cells (let' s call it flood fill)
func (g *Game) revealAdjacentCells(row, col int) {
	for dr := -1; dr <= 1; dr++ {
		for dc := -1; dc <= 1; dc++ {
			if dr == 0 && dc == 0 {
				continue
			}
			nr, nc := row+dr, col+dc
			if nr >= 0 && nr < g.Rows && nc >= 0 && nc < g.Cols {
				cell := &g.Board[nr][nc]
				if cell.State == StateHidden && !cell.IsMine {
					cell.State = StateRevealed
			     		g.CellsRevealed++
					if cell.AdjacentMines == 0 {
						g.revealAdjacentCells(nr, nc)
					}
				}
			}
		}
	}
}

// ToggleFlag toggles a flag on a cell
func (g *Game) ToggleFlag(row, col int) {
	// Validate coordinates
	if row < 0 || row >= g.Rows || col < 0 || col >= g.Cols {
		return
	}

	cell := &g.Board[row][col]

	// Can only flag hidden cells
	if cell.State == StateRevealed {
		return
	}

	if cell.State == StateFlagged {
		cell.State = StateHidden
		g.FlagsPlaced--
	} else {
		cell.State = StateFlagged
		g.FlagsPlaced++
	}
}

// revealAllMines reveals all mines (called when game is lost)
func (g *Game) revealAllMines() {
	for row := 0; row < g.Rows; row++ {
		for col := 0; col < g.Cols; col++ {
			if g.Board[row][col].IsMine {
				g.Board[row][col].State = StateRevealed
			} // of if!
		}
	}
}

// checkWinCondition checks if the player has won
func (g *Game) checkWinCondition() {
	totalCells := g.Rows * g.Cols
	if g.CellsRevealed == totalCells-g.TotalMines {
		g.Status = StatusWon
	}
}

// GetElapsedTime returns the elapsed time since the game started
func (g *Game) GetElapsedTime() time.Duration {
	if g.FirstMove {
		return 0
	}
	if g.Status != StatusPlaying {
		// Game is over, could track end time, but for now just return 0
		return 0
	}
	return time.Since(g.StartTime)
}
