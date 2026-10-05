package game

import "sort"

const LeaderboardSize = 20

type LeaderboardEntry struct {
	Name        string `json:"name"`
	TotalEarned int    `json:"totalEarned"`
	GamesWon    int    `json:"gamesWon"`
	Version     int    `json:"version"`
}

// Leaderboard is the complete state of the leaderboard.
type Leaderboard struct {
	Entries []LeaderboardEntry `json:"entries"`
}

type LeaderboardView struct {
	Entries []LeaderboardRank `json:"entries"`
}

type LeaderboardRank struct {
	Rank        int    `json:"rank"`
	Name        string `json:"name"`
	TotalEarned int    `json:"totalEarned"`
	GamesWon    int    `json:"gamesWon"`
}

// Upsert applies a snapshot. Snapshots older than the stored one are ignored,
// so applying the same snapshot twice changes nothing.
func (l *Leaderboard) Upsert(entry LeaderboardEntry) {
	found := false
	for i := range l.Entries {
		if l.Entries[i].Name != entry.Name {
			continue
		}
		if entry.Version <= l.Entries[i].Version {
			return
		}
		l.Entries[i] = entry
		found = true
	}
	if !found {
		l.Entries = append(l.Entries, entry)
	}
	sort.SliceStable(l.Entries, func(i, j int) bool {
		a, b := l.Entries[i], l.Entries[j]
		if a.TotalEarned != b.TotalEarned {
			return a.TotalEarned > b.TotalEarned
		}
		if a.GamesWon != b.GamesWon {
			return a.GamesWon > b.GamesWon
		}
		return a.Name < b.Name
	})
	if len(l.Entries) > LeaderboardSize {
		l.Entries = l.Entries[:LeaderboardSize]
	}
}

func (l *Leaderboard) View() LeaderboardView {
	ranks := make([]LeaderboardRank, len(l.Entries))
	for i, e := range l.Entries {
		ranks[i] = LeaderboardRank{Rank: i + 1, Name: e.Name, TotalEarned: e.TotalEarned, GamesWon: e.GamesWon}
	}
	return LeaderboardView{Entries: ranks}
}
