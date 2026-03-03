// Package fuzzy provides a fuzzy string matching algorithm with scoring,
// similar to the matching used in fzf. Results are ranked by quality so
// exact / prefix / tight matches always beat scattered ones.
package fuzzy

import (
	"strings"
	"unicode"
)

// Result holds the outcome of a fuzzy score computation.
type Result struct {
	// Matched is true when all pattern chars were found in order.
	Matched bool
	// Score ranks the quality of the match (higher is better).
	Score int
	// Positions holds the indices into the target string where each
	// pattern character was matched.
	Positions []int
}

// MinQuality returns the minimum score a match must reach to be considered
// relevant for a given pattern length. Matches below this threshold should
// be discarded by the caller.
func MinQuality(patternLen int) int {
	// A tight match (consecutive chars) scores ~24 per char (16 base + 8 consec).
	// We require at least ~30 per char so that scattered matches with big
	// gaps (which get penalised at -8/position) are filtered out.
	// For example, "htop" needs score >= 120 to pass.
	return patternLen * 30
}

// Score computes a fuzzy match score for pattern against target.
//
// Scoring heuristics (all case-insensitive):
//
//   - +16 for each matched character
//   - +8 bonus for consecutive matches
//   - +12 bonus for match at the very start of the string
//   - +10 bonus for match at a word boundary (after separator or case change)
//   - -8 penalty per position of gap between consecutive matched chars
//   - Exact substring match gets a large bonus (+100)
//   - Prefix match gets an even larger bonus (+200)
//   - Exact full-string match gets the highest bonus (+300)
//   - Short targets get a length bonus (prefer "htop" over "libhttp-foo")
//
// Returns a Result with Matched=false if the pattern doesn't match.
func Score(pattern, target string) Result {
	if pattern == "" {
		return Result{Matched: true, Score: 0}
	}

	p := strings.ToLower(pattern)
	t := strings.ToLower(target)

	res := greedyMatch(p, t, target)
	if !res.Matched {
		return res
	}
	
	// Exact full string
	if t == p {
		res.Score += 300
	}
	// Prefix match
	if strings.HasPrefix(t, p) {
		res.Score += 200
	}
	// Exact substring (contiguous)
	if strings.Contains(t, p) {
		res.Score += 100
	}
	// Length bonus: shorter targets rank higher for the same quality.
	// Max bonus 30 for targets shorter than pattern+4, tapering to 0
	// for very long targets.
	if lenDiff := len(t) - len(p); lenDiff < 30 {
		bonus := 30 - lenDiff
		if bonus < 0 {
			bonus = 0
		}
		res.Score += bonus
	}

	return res
}

// greedyMatch finds the first greedy alignment and computes a score.
func greedyMatch(pattern, targetLower, targetOrig string) Result {
	positions := make([]int, 0, len(pattern))
	score := 0
	pi := 0
	lastMatchIdx := -1

	for ti := 0; ti < len(targetLower) && pi < len(pattern); ti++ {
		if targetLower[ti] != pattern[pi] {
			continue
		}
		positions = append(positions, ti)

		// Base score per matched character
		score += 16

		if lastMatchIdx >= 0 {
			gap := ti - lastMatchIdx - 1
			if gap == 0 {
				// Consecutive bonus
				score += 8
			} else {
				// Heavy gap penalty — scattered matches are heavily punished
				score -= gap * 8
			}
		}

		// Start of string bonus
		if ti == 0 {
			score += 12
		}

		// Word boundary bonus: after separator or camelCase transition
		if ti > 0 {
			prev := rune(targetOrig[ti-1])
			cur := rune(targetOrig[ti])
			if isSeparator(prev) || (unicode.IsLower(prev) && unicode.IsUpper(cur)) {
				score += 10
			}
		}

		lastMatchIdx = ti
		pi++
	}

	if pi < len(pattern) {
		return Result{Matched: false}
	}
	return Result{Matched: true, Score: score, Positions: positions}
}

func isSeparator(r rune) bool {
	return r == ' ' || r == '-' || r == '_' || r == '.' || r == '/' || r == ':'
}
