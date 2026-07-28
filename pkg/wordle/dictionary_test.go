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
	"slices"
	"testing"
)

func TestValidWord(t *testing.T) {
	t.Parallel()

	valid := []string{"CRANE", "ZYMIC", "AAHED"}
	for _, w := range valid {
		if err := ValidWord(w); err != nil {
			t.Errorf("ValidWord(%q) = %v, want nil", w, err)
		}
	}

	invalid := []struct{ name, word string }{
		{"empty", ""},
		{"too short", "CRAN"},
		{"too long", "CRANES"},
		{"lowercase", "crane"},
		{"digit", "CR4NE"},
		{"space", "CR NE"},
		{"regex metacharacter", "CR.NE"},
		{"regex character class", "[A-Z]"},
		// Five bytes, four runes. A byte-length check alone lets this through,
		// and indexing it by byte yields fragments of a UTF-8 sequence.
		{"multi-byte rune", "CRÁN"},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := ValidWord(tc.word); !errors.Is(err, ErrInvalidWord) {
				t.Errorf("ValidWord(%q) = %v, want one wrapping ErrInvalidWord", tc.word, err)
			}
		})
	}
}

func TestNewDictionary(t *testing.T) {
	t.Parallel()

	t.Run("normalizes, sorts and deduplicates", func(t *testing.T) {
		t.Parallel()
		got, err := NewDictionary([]string{"toast", " CRANE ", "abide", "CRANE"})
		if err != nil {
			t.Fatalf("NewDictionary returned unexpected error: %v", err)
		}
		want := []string{"ABIDE", "CRANE", "TOAST"}
		if !slices.Equal(got.Words, want) {
			t.Errorf("Words = %v, want %v", got.Words, want)
		}
	})

	t.Run("rejects malformed entries", func(t *testing.T) {
		t.Parallel()
		if _, err := NewDictionary([]string{"CRANE", "TOO-LONG"}); !errors.Is(err, ErrInvalidWord) {
			t.Errorf("error = %v, want one wrapping ErrInvalidWord", err)
		}
	})

	t.Run("accepts an empty list", func(t *testing.T) {
		t.Parallel()
		got, err := NewDictionary(nil)
		if err != nil {
			t.Fatalf("NewDictionary(nil) returned unexpected error: %v", err)
		}
		if len(got.Words) != 0 {
			t.Errorf("Words = %v, want empty", got.Words)
		}
	})
}

func TestShippedDictionary(t *testing.T) {
	t.Parallel()

	dict := DefaultDictionary()

	// The shipped list is the standard Wordle allowed-guess set. Pinning the
	// count turns an accidental edit to words.go into a test failure rather
	// than a silent change in every suggestion the tool makes.
	const wantWords = 12972
	if got := len(dict.Words); got != wantWords {
		t.Errorf("len(Words) = %d, want %d", got, wantWords)
	}

	if !slices.IsSorted(dict.Words) {
		t.Error("Words is not sorted")
	}

	seen := make(map[string]struct{}, len(dict.Words))
	for _, w := range dict.Words {
		if err := ValidWord(w); err != nil {
			t.Fatalf("dictionary contains an invalid entry: %v", err)
		}
		if _, dup := seen[w]; dup {
			t.Errorf("duplicate entry %q", w)
		}
		seen[w] = struct{}{}
	}

	for _, w := range []string{"CRANE", "TOAST", "SASSY", "ZYMIC", "AAHED"} {
		if _, ok := seen[w]; !ok {
			t.Errorf("expected %q in the dictionary", w)
		}
	}
}

func TestShippedAnswers(t *testing.T) {
	t.Parallel()

	dict := DefaultDictionary()

	// Pinning the count turns an accidental edit to answers.go into a test
	// failure rather than a silent shift in every ranking.
	const wantAnswers = 2315
	if got := len(dict.answers); got != wantAnswers {
		t.Errorf("len(answers) = %d, want %d", got, wantAnswers)
	}

	// The answers must be a strict subset of the legal guesses. Nothing at run
	// time enforces that now the lists are Go slices, so this test is the only
	// thing standing between a bad edit and silently wrong rankings.
	for answer := range dict.answers {
		if !slices.Contains(dict.Words, answer) {
			t.Errorf("answer %q is not a legal guess", answer)
		}
	}

	if !dict.IsAnswer("TOAST") {
		t.Error("IsAnswer(\"TOAST\") = false, want true")
	}
	// Legal guesses that have never been solutions. Keeping these out of the
	// answer set is the entire point of shipping it.
	for _, word := range []string{"LOAST", "PLAST", "QAPIK", "ZYMIC"} {
		if dict.IsAnswer(word) {
			t.Errorf("IsAnswer(%q) = true, want false", word)
		}
	}
}

func TestAnswersFiltersInOrder(t *testing.T) {
	t.Parallel()

	dict := DefaultDictionary()

	got := dict.Answers([]string{"LOAST", "PLAST", "TOAST"})
	if want := []string{"TOAST"}; !slices.Equal(got, want) {
		t.Errorf("Answers = %v, want %v", got, want)
	}

	if got := dict.Answers(nil); len(got) != 0 {
		t.Errorf("Answers(nil) = %v, want empty", got)
	}
}

// TestFixtureDictionaryHasNoAnswers documents that a caller-supplied dictionary
// carries no answer list, so IsAnswer is false for everything rather than
// silently consulting the embedded one.
func TestFixtureDictionaryHasNoAnswers(t *testing.T) {
	t.Parallel()

	d, err := NewDictionary([]string{"TOAST", "CRANE"})
	if err != nil {
		t.Fatalf("NewDictionary returned unexpected error: %v", err)
	}
	if d.IsAnswer("TOAST") {
		t.Error("IsAnswer(\"TOAST\") = true on a fixture dictionary, want false")
	}
}

func TestDefaultDictionaryIsShared(t *testing.T) {
	t.Parallel()

	first := DefaultDictionary()
	second := DefaultDictionary()
	if first != second {
		t.Error("DefaultDictionary() returned distinct values, want the same shared instance")
	}
}
