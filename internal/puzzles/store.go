// Package puzzles reads puzzle files. It is provided by the course.
package puzzles

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"sync"

	"wordflow/internal/game"
)

var ErrNotFound = errors.New("puzzle not found")

var idPattern = regexp.MustCompile(`^[a-z0-9-]+$`)

// Store loads puzzles from a directory of JSON files.
//
// Set PUZZLES_FAIL_FIRST=N to make the first N calls to Get fail. Lesson 2
// uses this to show Activity retries.
type Store struct {
	Dir string

	mu        sync.Mutex
	failsLeft int
}

// Summary is what the puzzle picker shows.
type Summary struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	Words            int    `json:"words"`
	TimeLimitSeconds int    `json:"timeLimitSeconds"`
}

func NewStore() *Store {
	dir := os.Getenv("PUZZLES_DIR")
	if dir == "" {
		dir = "puzzles"
	}
	fails, _ := strconv.Atoi(os.Getenv("PUZZLES_FAIL_FIRST"))
	return &Store{Dir: dir, failsLeft: fails}
}

func (s *Store) Get(id string) (game.Puzzle, error) {
	s.mu.Lock()
	if s.failsLeft > 0 {
		s.failsLeft--
		s.mu.Unlock()
		return game.Puzzle{}, errors.New("puzzle storage is temporarily unavailable")
	}
	s.mu.Unlock()

	if !idPattern.MatchString(id) {
		return game.Puzzle{}, ErrNotFound
	}
	data, err := os.ReadFile(filepath.Join(s.Dir, id+".json"))
	if errors.Is(err, os.ErrNotExist) {
		return game.Puzzle{}, ErrNotFound
	}
	if err != nil {
		return game.Puzzle{}, err
	}
	var puzzle game.Puzzle
	if err := json.Unmarshal(data, &puzzle); err != nil {
		return game.Puzzle{}, fmt.Errorf("parse puzzle %s: %w", id, err)
	}
	return puzzle, nil
}

// Save writes a puzzle file. An existing file with the same ID is replaced.
func (s *Store) Save(p game.Puzzle) error {
	if !idPattern.MatchString(p.ID) {
		return fmt.Errorf("invalid puzzle id %q", p.ID)
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.Dir, p.ID+".json"), append(data, '\n'), 0o644)
}

// Exists reports whether a puzzle file exists. It never simulates failures.
func (s *Store) Exists(id string) bool {
	if !idPattern.MatchString(id) {
		return false
	}
	_, err := os.Stat(filepath.Join(s.Dir, id+".json"))
	return err == nil
}

func (s *Store) List() ([]Summary, error) {
	paths, err := filepath.Glob(filepath.Join(s.Dir, "*.json"))
	if err != nil {
		return nil, err
	}
	summaries := []Summary{}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var p game.Puzzle
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		summaries = append(summaries, Summary{ID: p.ID, Title: p.Title, Words: len(p.Words), TimeLimitSeconds: p.TimeLimitSeconds})
	}
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Words < summaries[j].Words || (summaries[i].Words == summaries[j].Words && summaries[i].ID < summaries[j].ID)
	})
	return summaries, nil
}

// AllWords returns every answer and extra word across all puzzles.
func (s *Store) AllWords() (map[string]bool, error) {
	paths, err := filepath.Glob(filepath.Join(s.Dir, "*.json"))
	if err != nil {
		return nil, err
	}
	words := map[string]bool{}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var p game.Puzzle
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, err
		}
		for _, w := range append(p.Words, p.Extras...) {
			words[w] = true
		}
	}
	return words, nil
}
