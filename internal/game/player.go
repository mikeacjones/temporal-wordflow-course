package game

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

const StartingPoints = 20

var namePattern = regexp.MustCompile(`^[a-z0-9_]{3,20}$`)

// Player is the complete state of one player.
type Player struct {
	Name         string    `json:"name"`
	JoinedAt     time.Time `json:"joinedAt"`
	Points       int       `json:"points"`
	TotalEarned  int       `json:"totalEarned"`
	GamesStarted int       `json:"gamesStarted"`
	GamesPlayed  int       `json:"gamesPlayed"`
	GamesWon     int       `json:"gamesWon"`
	ActiveGameID string    `json:"activeGameId,omitempty"`
}

type PlayerView struct {
	Name         string    `json:"name"`
	JoinedAt     time.Time `json:"joinedAt"`
	Points       int       `json:"points"`
	TotalEarned  int       `json:"totalEarned"`
	GamesPlayed  int       `json:"gamesPlayed"`
	GamesWon     int       `json:"gamesWon"`
	ActiveGameID string    `json:"activeGameId,omitempty"`
}

func NormalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func ValidateName(name string) error {
	if !namePattern.MatchString(name) {
		return Error("invalid_name", "names are 3-20 characters: a-z, 0-9, or _")
	}
	return nil
}

func NewPlayer(name string, now time.Time) *Player {
	return &Player{Name: name, JoinedAt: now, Points: StartingPoints}
}

func (p *Player) CheckStartGame() error {
	if p.ActiveGameID != "" {
		return Error("game_active", "finish your current game first")
	}
	return nil
}

// NextGameID is the Workflow ID for this player's next game.
func (p *Player) NextGameID() string {
	return fmt.Sprintf("game-%s-%d", p.Name, p.GamesStarted+1)
}

func (p *Player) StartGame(gameID string) {
	p.GamesStarted++
	p.ActiveGameID = gameID
}

// FinishGame records a finished game. It reports whether the result applied.
func (p *Player) FinishGame(result Result) bool {
	if result.GameID != p.ActiveGameID {
		return false
	}
	p.ActiveGameID = ""
	p.GamesPlayed++
	if result.Won {
		p.GamesWon++
	}
	p.Points += result.Score
	p.TotalEarned += result.Score
	return true
}

// ClearGame drops the active game without scoring it.
func (p *Player) ClearGame(gameID string) {
	if p.ActiveGameID == gameID {
		p.ActiveGameID = ""
	}
}

func (p *Player) CheckSpend(gameID string, amount int) error {
	if amount <= 0 {
		return Error("invalid_amount", "amount must be positive")
	}
	if gameID != p.ActiveGameID {
		return Error("game_not_active", "that game is not this player's active game")
	}
	if p.Points < amount {
		return Error("insufficient_points", fmt.Sprintf("not enough points: need %d, have %d", amount, p.Points))
	}
	return nil
}

func (p *Player) Spend(amount int) {
	p.Points -= amount
}

func (p *Player) CheckWithdraw(amount int) error {
	if amount <= 0 {
		return Error("invalid_amount", "amount must be positive")
	}
	if p.Points < amount {
		return Error("insufficient_points", fmt.Sprintf("not enough points: need %d, have %d", amount, p.Points))
	}
	return nil
}

// Withdraw removes spendable points, for example to gift them.
func (p *Player) Withdraw(amount int) {
	p.Points -= amount
}

func (p *Player) CheckCredit(amount int) error {
	if amount <= 0 {
		return Error("invalid_amount", "amount must be positive")
	}
	return nil
}

// Credit adds spendable points that were not earned by playing.
func (p *Player) Credit(amount int) {
	p.Points += amount
}

func (p *Player) View() PlayerView {
	return PlayerView{
		Name:         p.Name,
		JoinedAt:     p.JoinedAt,
		Points:       p.Points,
		TotalEarned:  p.TotalEarned,
		GamesPlayed:  p.GamesPlayed,
		GamesWon:     p.GamesWon,
		ActiveGameID: p.ActiveGameID,
	}
}

// LeaderboardEntry is an absolute snapshot. Version only increases.
func (p *Player) LeaderboardEntry() LeaderboardEntry {
	return LeaderboardEntry{Name: p.Name, TotalEarned: p.TotalEarned, GamesWon: p.GamesWon, Version: p.GamesPlayed}
}
