// Copyright 2022 Mike Helmick
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package wordle

import (
	"errors"
	"fmt"
	"strings"
)

// Feedback characters, as they appear in a pattern string.
const (
	Absent  byte = '.' // letter is not in the word (gray)
	Present byte = 'Y' // letter is in the word, but not here (yellow)
	Correct byte = 'G' // letter is in the word, right here (green)
)

var (
	// ErrInvalidPattern is returned for a pattern that is not exactly WordLen
	// characters drawn from Absent, Present and Correct.
	ErrInvalidPattern = errors.New("invalid pattern")

	// ErrContradiction is returned when a turn's feedback cannot be reconciled
	// with what is already known, which in practice means the feedback was
	// entered incorrectly.
	ErrContradiction = errors.New("contradictory feedback")
)

// Knowledge is everything deduced about the hidden word so far. The zero value
// is not usable; call NewKnowledge.
//
// State accumulates across turns. That is the whole point of the type, and the
// reason the fields are unexported: an earlier version kept the green and
// yellow constraints in caller-owned fields that the CLI reset on every round,
// so confirmed letters were silently forgotten and impossible words came back
// as suggestions.
type Knowledge struct {
	// greens[i] is the letter known to sit at position i, or 0 if unknown.
	greens [WordLen]byte
	// banned[i] has bit (letter-'A') set when that letter cannot sit at
	// position i.
	banned [WordLen]uint32
	// minCount and maxCount bound how many times each letter appears.
	minCount [alphabetSize]uint8
	maxCount [alphabetSize]uint8
	// guessed holds words already played, which are never suggested again.
	guessed map[string]struct{}
	// solved is the word whose feedback came back all green, or "".
	//
	// Worth recording separately because the winning word disappears from the
	// suggestions: it is a played word, so it is filtered out like any other,
	// and a solved puzzle looks exactly like an unsatisfiable one -- zero
	// candidates. Without this a caller cannot tell "you won" from "you mistyped
	// a colour somewhere".
	solved string
}

// NewKnowledge returns Knowledge with no constraints: every word matches.
func NewKnowledge() *Knowledge {
	k := &Knowledge{guessed: make(map[string]struct{})}
	for i := range k.maxCount {
		k.maxCount[i] = WordLen
	}
	return k
}

// ValidPattern reports whether p is a well-formed feedback string.
func ValidPattern(p string) error {
	if len(p) != WordLen {
		return fmt.Errorf("%w: %q is %d bytes, want %d", ErrInvalidPattern, p, len(p), WordLen)
	}
	for i := range len(p) {
		switch p[i] {
		case Absent, Present, Correct:
		default:
			return fmt.Errorf("%w: %q has %q at position %d, want one of %q, %q or %q",
				ErrInvalidPattern, p, string(p[i]), i+1, string(Absent), string(Present), string(Correct))
		}
	}
	return nil
}

// Apply folds one turn of feedback into k.
//
// guess is the word played and pattern is its per-position result, e.g.
// Apply("CRANE", ".YG..") for a guess where R is somewhere else and A is
// correctly placed. Both are upper-cased and validated, so caller data never
// reaches the matcher unchecked.
//
// If the feedback contradicts what is already known, Apply returns an error
// wrapping ErrContradiction and leaves k unmodified.
func (k *Knowledge) Apply(guess, pattern string) error {
	guess = strings.ToUpper(strings.TrimSpace(guess))
	pattern = strings.ToUpper(strings.TrimSpace(pattern))
	if err := ValidWord(guess); err != nil {
		return fmt.Errorf("guess: %w", err)
	}
	if err := ValidPattern(pattern); err != nil {
		return err
	}

	// Stage every deduction on copies so that returning an error partway
	// through leaves k exactly as it was.
	greens := k.greens
	banned := k.banned
	minCount := k.minCount
	maxCount := k.maxCount

	// seen counts green and yellow tiles per letter in this guess; absent
	// counts gray ones. Both are per-turn: the bounds they imply are only
	// valid within a single, self-consistent piece of feedback.
	var seen, absent [alphabetSize]uint8

	for i := range WordLen {
		c := guess[i]
		l := c - 'A'
		switch pattern[i] {
		case Correct:
			if prev := greens[i]; prev != 0 && prev != c {
				return fmt.Errorf("%w: position %d is already known to be %q, it cannot also be %q",
					ErrContradiction, i+1, string(prev), string(c))
			}
			if banned[i]&(1<<l) != 0 {
				return fmt.Errorf("%w: %q is already known not to be at position %d",
					ErrContradiction, string(c), i+1)
			}
			greens[i] = c
			seen[l]++
		case Present:
			if greens[i] == c {
				return fmt.Errorf("%w: %q is already known to be at position %d, so it cannot be misplaced there",
					ErrContradiction, string(c), i+1)
			}
			banned[i] |= 1 << l
			seen[l]++
		case Absent:
			if greens[i] == c {
				return fmt.Errorf("%w: %q is already known to be at position %d, so it cannot be absent there",
					ErrContradiction, string(c), i+1)
			}
			banned[i] |= 1 << l
			absent[l]++
		}
	}

	for l := range alphabetSize {
		if seen[l] > minCount[l] {
			minCount[l] = seen[l]
		}
		// A gray tile proves the word holds no more of that letter than the
		// green and yellow tiles in the same guess already account for. This is
		// what lets a guess of SASSY against MOSSY teach "exactly two S", not
		// merely "at least one S" -- and it is the signal the previous
		// implementation discarded when it stripped such letters wholesale from
		// its exclusion set.
		if absent[l] > 0 && seen[l] < maxCount[l] {
			maxCount[l] = seen[l]
		}
		if minCount[l] > maxCount[l] {
			return fmt.Errorf("%w: %q would have to appear at least %d and at most %d times",
				ErrContradiction, string(rune('A'+l)), minCount[l], maxCount[l])
		}
	}

	k.greens = greens
	k.banned = banned
	k.minCount = minCount
	k.maxCount = maxCount
	k.guessed[guess] = struct{}{}
	if strings.Count(pattern, string(Correct)) == WordLen {
		k.solved = guess
	}
	return nil
}

// Match reports whether word is consistent with everything k knows.
//
// A word that is not exactly WordLen uppercase ASCII letters never matches.
func (k *Knowledge) Match(word string) bool {
	if len(word) != WordLen {
		return false
	}

	var counts [alphabetSize]uint8
	for i := range WordLen {
		c := word[i]
		if c < 'A' || c > 'Z' {
			return false
		}
		if g := k.greens[i]; g != 0 && g != c {
			return false
		}
		l := c - 'A'
		if k.banned[i]&(1<<l) != 0 {
			return false
		}
		counts[l]++
	}

	for l := range alphabetSize {
		if counts[l] < k.minCount[l] || counts[l] > k.maxCount[l] {
			return false
		}
	}

	_, alreadyGuessed := k.guessed[word]
	return !alreadyGuessed
}

// Solved returns the winning word and true once a turn has come back all green.
//
// Callers should check this before reporting on the candidate list, which is
// empty in exactly this case.
func (k *Knowledge) Solved() (string, bool) {
	return k.solved, k.solved != ""
}

// Guessed reports whether word has already been played.
func (k *Knowledge) Guessed(word string) bool {
	_, ok := k.guessed[strings.ToUpper(strings.TrimSpace(word))]
	return ok
}

// String renders k for debugging: the known letters as a positional mask,
// followed by any letter whose count is constrained.
func (k *Knowledge) String() string {
	var b strings.Builder
	b.WriteString("greens=")
	for i := range WordLen {
		if k.greens[i] == 0 {
			b.WriteByte(Absent)
		} else {
			b.WriteByte(k.greens[i])
		}
	}

	b.WriteString(" counts=[")
	first := true
	for l := range alphabetSize {
		if k.minCount[l] == 0 && k.maxCount[l] == WordLen {
			continue
		}
		if !first {
			b.WriteByte(' ')
		}
		first = false
		fmt.Fprintf(&b, "%c:%d-%d", 'A'+l, k.minCount[l], k.maxCount[l])
	}
	b.WriteByte(']')

	b.WriteString(" banned=[")
	first = true
	for i := range WordLen {
		if k.banned[i] == 0 {
			continue
		}
		if !first {
			b.WriteByte(' ')
		}
		first = false
		fmt.Fprintf(&b, "%d:", i+1)
		for l := range alphabetSize {
			if k.banned[i]&(1<<uint(l)) != 0 {
				b.WriteByte(byte('A' + l))
			}
		}
	}
	b.WriteByte(']')

	fmt.Fprintf(&b, " guessed=%d", len(k.guessed))
	if k.solved != "" {
		fmt.Fprintf(&b, " solved=%s", k.solved)
	}
	return b.String()
}
