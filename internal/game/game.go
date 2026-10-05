// Package game contains the Wordflow rules. It is provided by the course.
//
// Everything here is deterministic: no I/O, no clocks, no randomness. Callers
// pass the current time in. That makes it safe to call from Workflow code.
package game

import (
	"slices"
	"strings"
	"time"

	"go.temporal.io/sdk/temporal"
)

const (
	MinWordLength    = 3
	FreeHints        = 2
	HintCost         = 10
	ExtensionSeconds = 30
	MaxExtensions    = 2
	WinBonus         = 10
	BonusWordPoints  = 2
)

const (
	StatusLoading  = "loading"
	StatusPlaying  = "playing"
	StatusWon      = "won"
	StatusTimedOut = "timed_out"
)

const (
	OutcomeFound        = "found"
	OutcomeAlreadyFound = "already_found"
	OutcomeNotInPuzzle  = "not_in_puzzle"
	OutcomeBonus        = "bonus"
	OutcomeInvalid      = "invalid"
	OutcomeTimedOut     = "timed_out"
	OutcomeGameOver     = "game_over"
)

type Puzzle struct {
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	Letters          string   `json:"letters"`
	TimeLimitSeconds int      `json:"timeLimitSeconds"`
	Words            []string `json:"words"`
	Extras           []string `json:"extras,omitempty"`
}

// TimeLimit is the puzzle's starting time limit.
func (p Puzzle) TimeLimit() time.Duration {
	return time.Duration(p.TimeLimitSeconds) * time.Second
}

// State is the complete state of one game.
type State struct {
	GameID     string    `json:"gameId"`
	PlayerID   string    `json:"playerId,omitempty"`
	Puzzle     Puzzle    `json:"puzzle"`
	Status     string    `json:"status"`
	StartedAt  time.Time `json:"startedAt"`
	Deadline   time.Time `json:"deadline,omitempty"`
	FinishedAt time.Time `json:"finishedAt,omitempty"`
	Found      []string  `json:"found"`
	Revealed   []int     `json:"revealed"`
	Rejected   []string  `json:"rejected"`
	BonusWords []string  `json:"bonusWords"`
	Guesses    int       `json:"guesses"`
	Incorrect  int       `json:"incorrect"`
	HintsUsed  int       `json:"hintsUsed"`
	Extensions int       `json:"extensions"`
	Score      int       `json:"score"`
}

func NewState(gameID, playerID string, puzzle Puzzle, now time.Time) *State {
	return &State{
		GameID:     gameID,
		PlayerID:   playerID,
		Puzzle:     puzzle,
		Status:     StatusPlaying,
		StartedAt:  now,
		Found:      []string{},
		Revealed:   make([]int, len(puzzle.Words)),
		Rejected:   []string{},
		BonusWords: []string{},
	}
}

func Normalize(word string) string {
	return strings.ToUpper(strings.TrimSpace(word))
}

// Over reports whether the game has finished.
func (s *State) Over() bool {
	return s.Status == StatusWon || s.Status == StatusTimedOut
}

// CheckGuess returns an error if the word can never be a valid guess.
func (s *State) CheckGuess(word string) error {
	word = Normalize(word)
	if s.Over() {
		return Error("game_over", "the game is over")
	}
	if len(word) < MinWordLength {
		return Error("too_short", "words need at least 3 letters")
	}
	if !canSpell(word, s.Puzzle.Letters) {
		return Error("invalid_letters", "use only the letters "+s.Puzzle.Letters)
	}
	return nil
}

// Guess applies a guess and returns one of the Outcome constants.
func (s *State) Guess(word string, now time.Time) string {
	word = Normalize(word)
	if s.Expire(now) {
		return OutcomeTimedOut
	}
	if s.Over() {
		return OutcomeGameOver
	}
	if s.CheckGuess(word) != nil {
		return OutcomeInvalid
	}

	s.Guesses++
	if slices.Contains(s.Found, word) {
		return OutcomeAlreadyFound
	}
	if slices.Contains(s.Puzzle.Words, word) {
		s.Found = append(s.Found, word)
		s.finishIfSolved(now)
		return OutcomeFound
	}
	s.Incorrect++
	if !slices.Contains(s.Rejected, word) {
		s.Rejected = append([]string{word}, s.Rejected...)
	}
	return OutcomeNotInPuzzle
}

// MarkBonus turns a rejected guess into a bonus word.
func (s *State) MarkBonus(word string) {
	word = Normalize(word)
	if slices.Contains(s.BonusWords, word) {
		return
	}
	if i := slices.Index(s.Rejected, word); i >= 0 {
		s.Rejected = slices.Delete(s.Rejected, i, i+1)
		s.Incorrect--
	}
	s.BonusWords = append(s.BonusWords, word)
}

// NextHintCost is 0 while free hints remain, then HintCost.
func (s *State) NextHintCost() int {
	if s.HintsUsed < FreeHints {
		return 0
	}
	return HintCost
}

func (s *State) CheckHint() error {
	if s.Over() {
		return Error("game_over", "the game is over")
	}
	if s.NextHintCost() > 0 && s.PlayerID == "" {
		return Error("no_free_hints", "guests only get 2 free hints; join as a player to buy more")
	}
	return nil
}

// RevealLetter reveals the next hidden letter of the first unsolved word.
// A word with every letter revealed counts as found.
func (s *State) RevealLetter(now time.Time) error {
	if err := s.CheckHint(); err != nil {
		return err
	}
	for i, word := range s.Puzzle.Words {
		if slices.Contains(s.Found, word) {
			continue
		}
		s.Revealed[i]++
		s.HintsUsed++
		if s.Revealed[i] >= len(word) {
			s.Found = append(s.Found, word)
			s.finishIfSolved(now)
		}
		return nil
	}
	return Error("nothing_to_reveal", "every word is already found")
}

// SetDeadline starts the game clock.
func (s *State) SetDeadline(deadline time.Time) {
	s.Deadline = deadline
}

func (s *State) CheckExtend() error {
	if s.Over() {
		return Error("game_over", "the game is over")
	}
	if s.Deadline.IsZero() {
		return Error("no_clock", "this game has no time limit")
	}
	if s.Extensions >= MaxExtensions {
		return Error("no_extensions", "no time extensions left")
	}
	return nil
}

// Extend moves the deadline back by ExtensionSeconds.
func (s *State) Extend(now time.Time) error {
	s.Expire(now)
	if err := s.CheckExtend(); err != nil {
		return err
	}
	s.Deadline = s.Deadline.Add(ExtensionSeconds * time.Second)
	s.Extensions++
	return nil
}

// Expire ends the game if its deadline has passed. It reports whether the
// game is timed out.
func (s *State) Expire(now time.Time) bool {
	if s.Status == StatusPlaying && !s.Deadline.IsZero() && !now.Before(s.Deadline) {
		s.finish(StatusTimedOut, now)
	}
	return s.Status == StatusTimedOut
}

func (s *State) finishIfSolved(now time.Time) {
	if len(s.Found) == len(s.Puzzle.Words) {
		s.finish(StatusWon, now)
	}
}

func (s *State) finish(status string, now time.Time) {
	s.Status = status
	s.FinishedAt = now
	score := len(s.BonusWords)*BonusWordPoints - s.Incorrect
	for _, word := range s.Found {
		score += len(word)
	}
	if status == StatusWon {
		score += WinBonus
	}
	s.Score = max(0, score)
}

func canSpell(word, letters string) bool {
	available := map[rune]int{}
	for _, r := range strings.ToUpper(letters) {
		available[r]++
	}
	for _, r := range word {
		available[r]--
		if available[r] < 0 {
			return false
		}
	}
	return true
}

// Error returns a non-retryable Temporal ApplicationError. Returning one from
// an Update handler or validator rejects the Update with this message.
func Error(code, message string) error {
	return temporal.NewNonRetryableApplicationError(message, code, nil)
}
