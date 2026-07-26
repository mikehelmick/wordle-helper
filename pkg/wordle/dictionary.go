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
	"embed"
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

const wordFile = "words5.txt"

//go:embed words5.txt
var embedded embed.FS

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
	Words []string
}

// NewDictionary builds a Dictionary from an explicit word list. Words are
// upper-cased and sorted; every entry must satisfy ValidWord.
//
// Tests use this to run against a small fixture instead of the full embedded
// list.
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

// New parses the embedded word list into a new Dictionary.
//
// Prefer DefaultDictionary unless you specifically need an independent copy.
func New() (*Dictionary, error) {
	data, err := embedded.ReadFile(wordFile)
	if err != nil {
		return nil, fmt.Errorf("unable to load %s: %w", wordFile, err)
	}

	// The file ships with CRLF line endings and a trailing newline, so blank
	// entries are expected and skipped rather than treated as corruption.
	lines := strings.Split(string(data), "\n")
	words := make([]string, 0, len(lines))
	for i, line := range lines {
		word := strings.ToUpper(strings.TrimSpace(line))
		if word == "" {
			continue
		}
		if err := ValidWord(word); err != nil {
			return nil, fmt.Errorf("%s line %d: %w", wordFile, i+1, err)
		}
		words = append(words, word)
	}
	return NewDictionary(words)
}

var (
	defaultOnce sync.Once
	defaultDict *Dictionary
	defaultErr  error
)

// DefaultDictionary returns the shared Dictionary built from the embedded word
// list, parsing it on first use.
//
// This replaces an init() that panicked on failure. Importing a package should
// never be able to take down the process, and deferring the work also keeps
// callers that supply their own Dictionary from paying for a parse they will
// not use.
func DefaultDictionary() (*Dictionary, error) {
	defaultOnce.Do(func() {
		defaultDict, defaultErr = New()
	})
	return defaultDict, defaultErr
}
