package game

import (
	"slices"
	"strings"
	"time"
)

// View is what the web UI renders for a game.
type View struct {
	GameID         string     `json:"gameId"`
	PlayerID       string     `json:"playerId,omitempty"`
	PuzzleID       string     `json:"puzzleId"`
	Title          string     `json:"title"`
	Letters        string     `json:"letters"`
	Status         string     `json:"status"`
	Words          []WordView `json:"words"`
	FoundCount     int        `json:"foundCount"`
	TotalWords     int        `json:"totalWords"`
	Rejected       []string   `json:"rejected"`
	BonusWords     []string   `json:"bonusWords"`
	Guesses        int        `json:"guesses"`
	Incorrect      int        `json:"incorrect"`
	FreeHints      int        `json:"freeHints"`
	HintCost       int        `json:"hintCost"`
	ExtensionsLeft int        `json:"extensionsLeft"`
	ExpiresAt      *time.Time `json:"expiresAt"`
	Score          int        `json:"score"`
}

// WordView is one answer slot. Pattern shows known letters and "_" for hidden ones.
type WordView struct {
	Length  int    `json:"length"`
	Pattern string `json:"pattern"`
	Found   bool   `json:"found"`
}

type GuessResult struct {
	Word    string `json:"word"`
	Outcome string `json:"outcome"`
	Game    View   `json:"game"`
}

type HintResult struct {
	PointsSpent int  `json:"pointsSpent"`
	Game        View `json:"game"`
}

// Result is the small summary a finished game returns.
type Result struct {
	GameID     string    `json:"gameId"`
	PlayerID   string    `json:"playerId,omitempty"`
	PuzzleID   string    `json:"puzzleId"`
	Won        bool      `json:"won"`
	Score      int       `json:"score"`
	FinishedAt time.Time `json:"finishedAt"`
}

// LoadingView is shown before the puzzle has loaded.
func LoadingView(gameID string) View {
	return View{GameID: gameID, Status: StatusLoading, Words: []WordView{}, Rejected: []string{}, BonusWords: []string{}}
}

func (s *State) View() View {
	words := make([]WordView, len(s.Puzzle.Words))
	for i, word := range s.Puzzle.Words {
		found := slices.Contains(s.Found, word)
		shown := s.Revealed[i]
		if found || s.Status == StatusTimedOut {
			shown = len(word)
		}
		words[i] = WordView{
			Length:  len(word),
			Pattern: word[:shown] + strings.Repeat("_", len(word)-shown),
			Found:   found,
		}
	}

	view := View{
		GameID:         s.GameID,
		PlayerID:       s.PlayerID,
		PuzzleID:       s.Puzzle.ID,
		Title:          s.Puzzle.Title,
		Letters:        s.Puzzle.Letters,
		Status:         s.Status,
		Words:          words,
		FoundCount:     len(s.Found),
		TotalWords:     len(s.Puzzle.Words),
		Rejected:       slices.Clone(s.Rejected),
		BonusWords:     slices.Clone(s.BonusWords),
		Guesses:        s.Guesses,
		Incorrect:      s.Incorrect,
		FreeHints:      max(0, FreeHints-s.HintsUsed),
		HintCost:       s.NextHintCost(),
		ExtensionsLeft: MaxExtensions - s.Extensions,
		Score:          s.Score,
	}
	if !s.Deadline.IsZero() {
		deadline := s.Deadline
		view.ExpiresAt = &deadline
	}
	return view
}

func (s *State) Result() Result {
	return Result{
		GameID:     s.GameID,
		PlayerID:   s.PlayerID,
		PuzzleID:   s.Puzzle.ID,
		Won:        s.Status == StatusWon,
		Score:      s.Score,
		FinishedAt: s.FinishedAt,
	}
}
