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
	"slices"
	"strings"
	"sync"
)

// WordLen is the number of letters in every word this package handles.
const WordLen = 5

// alphabetSize is the number of distinct letters a word may be built from.
const alphabetSize = 26

// ErrInvalidWord is returned for any string that is not exactly WordLen
// uppercase ASCII letters. Callers can test for it with errors.Is.
var ErrInvalidWord = errors.New("invalid word")

// ValidWord reports whether s is exactly WordLen uppercase ASCII letters.
//
// The byte-length check and the A-Z check are deliberately paired: len(s) is a
// byte count, so on its own it would admit a multi-byte UTF-8 string that has
// fewer than WordLen runes. Indexing such a string by byte lands in the middle
// of an encoded rune and yields a garbage "letter", so every entry point that
// accepts caller data runs this first.
func ValidWord(s string) error {
	if len(s) != WordLen {
		return fmt.Errorf("%w: %q is %d bytes, want %d", ErrInvalidWord, s, len(s), WordLen)
	}
	for i := range len(s) {
		if s[i] < 'A' || s[i] > 'Z' {
			return fmt.Errorf("%w: %q has a non-letter at position %d", ErrInvalidWord, s, i)
		}
	}
	return nil
}

// Dictionary is an immutable, sorted, deduplicated set of candidate words.
type Dictionary struct {
	// Words is every word that may be guessed, and therefore every word the
	// filter considers.
	Words []string

	// answers is the subset of Words that has ever been used as a solution.
	//
	// Roughly one word in six. The rest exist only so the game accepts them as
	// input -- LOAST and PLAST are legal guesses that will never be the answer.
	// Filtering them out entirely would be a mistake, since the puzzle's
	// answers are now editorially chosen rather than drawn from a fixed list,
	// so a word outside this set can and does turn up. They are ranked rather
	// than removed: everything consistent with the feedback is still reported,
	// but the plausible words lead.
	answers map[string]struct{}
}

// IsAnswer reports whether word has ever been used as a solution, which makes
// it a plausible answer rather than merely a legal guess.
//
// Always false for a Dictionary built by NewDictionary, which has no answer
// list of its own.
func (d *Dictionary) IsAnswer(word string) bool {
	_, ok := d.answers[word]
	return ok
}

// Answers returns the plausible answers among words, preserving order.
func (d *Dictionary) Answers(words []string) []string {
	out := make([]string, 0, len(words))
	for _, w := range words {
		if d.IsAnswer(w) {
			out = append(out, w)
		}
	}
	return out
}

// NewDictionary builds a Dictionary from an explicit word list. Words are
// upper-cased and sorted; every entry must satisfy ValidWord.
//
// Tests use this to run against a small fixture instead of the full list. The
// result has no answer list, so IsAnswer is false for everything.
func NewDictionary(words []string) (*Dictionary, error) {
	out := make([]string, 0, len(words))
	seen := make(map[string]struct{}, len(words))
	for _, w := range words {
		w = strings.ToUpper(strings.TrimSpace(w))
		if err := ValidWord(w); err != nil {
			return nil, err
		}
		if _, dup := seen[w]; dup {
			continue
		}
		seen[w] = struct{}{}
		out = append(out, w)
	}
	slices.Sort(out)
	return &Dictionary{Words: out}, nil
}

var (
	defaultOnce sync.Once
	defaultDict *Dictionary
)

// DefaultDictionary returns the shared Dictionary of every legal guess, with
// the answer list attached.
//
// It cannot fail and so returns no error. The word lists are Go slices compiled
// into the binary rather than data parsed at run time, which means the only way
// for them to be malformed is for the package not to build. What would
// otherwise be run-time validation is asserted by the tests instead.
//
// The work is still deferred rather than done in init(): a package that builds
// maps at import time makes every importer pay for them, and this one has
// callers that supply their own Dictionary.
func DefaultDictionary() *Dictionary {
	defaultOnce.Do(func() {
		defaultDict = &Dictionary{
			Words:   words,
			answers: make(map[string]struct{}, len(answers)),
		}
		for _, answer := range answers {
			defaultDict.answers[answer] = struct{}{}
		}
	})
	return defaultDict
}
