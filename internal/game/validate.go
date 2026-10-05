package game

import (
	"fmt"
	"regexp"
	"slices"
)

var puzzleIDPattern = regexp.MustCompile(`^[a-z0-9-]{3,30}$`)

// ValidatePuzzle checks a submitted puzzle's shape. It does not check that
// the words are real words.
func ValidatePuzzle(p Puzzle) error {
	switch {
	case !puzzleIDPattern.MatchString(p.ID):
		return Error("invalid_puzzle", "id must be 3-30 characters: a-z, 0-9, or -")
	case p.Title == "":
		return Error("invalid_puzzle", "title is required")
	case len(p.Letters) < 4 || len(p.Letters) > 8:
		return Error("invalid_puzzle", "use 4-8 letters")
	case p.TimeLimitSeconds < 30:
		return Error("invalid_puzzle", "time limit must be at least 30 seconds")
	case len(p.Words) < 3:
		return Error("invalid_puzzle", "add at least 3 words")
	}
	for i, word := range p.Words {
		if word != Normalize(word) || len(word) < MinWordLength || !canSpell(word, p.Letters) {
			return Error("invalid_puzzle", fmt.Sprintf("%q must be 3+ uppercase letters from %s", word, p.Letters))
		}
		if slices.Contains(p.Words[:i], word) {
			return Error("invalid_puzzle", fmt.Sprintf("%q is listed twice", word))
		}
	}
	return nil
}
