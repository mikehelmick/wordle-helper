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
	"testing"
)

func TestFeedback(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		guess  string
		answer string
		want   string
	}{
		{"nothing in common", "CHIRP", "SLUMS", "....."},
		{"exact match", "CRANE", "CRANE", "GGGGG"},
		{"single green", "CRANE", "TOAST", "..G.."},
		{"single yellow", "CRANE", "SASSY", "..Y.."},
		{
			// The guess has two Es but the answer only one, so the first E
			// consumes it and the second is reported absent.
			name: "duplicate in guess, single in answer", guess: "SPEED", answer: "ABIDE", want: "..Y.Y",
		},
		{
			// Both of the answer's Es are available, so both of the guess's Es
			// are misplaced rather than absent.
			name: "duplicate in guess, duplicate in answer", guess: "SPEED", answer: "ERASE", want: "Y.YY.",
		},
		{
			// Greens claim their letters before any yellow is assigned: MOSSY's
			// two Ss are both green, leaving none for the leading S.
			name: "greens consume the available duplicates", guess: "SASSY", answer: "MOSSY", want: "..GGG",
		},
		{"duplicate split across yellow and green", "ALLEY", "LEVEL", ".YYG."},
		{"lowercase input is accepted", "crane", "toast", "..G.."},
		{"surrounding whitespace is trimmed", " CRANE ", " TOAST ", "..G.."},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := Feedback(tc.guess, tc.answer)
			if err != nil {
				t.Fatalf("Feedback(%q, %q) returned unexpected error: %v", tc.guess, tc.answer, err)
			}
			if got != tc.want {
				t.Errorf("Feedback(%q, %q) = %q, want %q", tc.guess, tc.answer, got, tc.want)
			}
		})
	}
}

// TestFeedbackIsSelfConsistent checks the property that actually matters: the
// answer always survives its own feedback. If Feedback and Match ever disagree
// about duplicate letters, this catches it without anyone having to hand-derive
// the expected pattern.
func TestFeedbackIsSelfConsistent(t *testing.T) {
	t.Parallel()

	dict, err := DefaultDictionary()
	if err != nil {
		t.Fatalf("DefaultDictionary() returned unexpected error: %v", err)
	}

	// Sampling keeps this to a few thousand pairs while still covering the
	// whole alphabet, including the awkward doubled-letter words.
	const stride = 407
	for i := 0; i < len(dict.Words); i += stride {
		answer := dict.Words[i]
		for j := 0; j < len(dict.Words); j += stride {
			guess := dict.Words[j]

			pattern, err := Feedback(guess, answer)
			if err != nil {
				t.Fatalf("Feedback(%q, %q) returned unexpected error: %v", guess, answer, err)
			}

			k := NewKnowledge()
			if err := k.Apply(guess, pattern); err != nil {
				t.Fatalf("Apply(%q, %q) for answer %q returned unexpected error: %v", guess, pattern, answer, err)
			}
			if guess == answer {
				continue // The answer is now a previous guess, so it cannot match.
			}
			if !k.Match(answer) {
				t.Errorf("answer %q does not match its own feedback from guess %q (pattern %q); knowledge: %v",
					answer, guess, pattern, k)
			}
		}
	}
}

func TestFeedbackRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		guess  string
		answer string
	}{
		{"short guess", "CRAN", "TOAST"},
		{"long guess", "CRANES", "TOAST"},
		{"short answer", "CRANE", "TOAS"},
		{"digits in guess", "CR4NE", "TOAST"},
		{"punctuation in guess", "CR.NE", "TOAST"},
		{"empty guess", "", "TOAST"},
		// Five bytes but only four runes: byte indexing would split the É.
		{"multi-byte rune", "CRÁN", "TOAST"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := Feedback(tc.guess, tc.answer); !errors.Is(err, ErrInvalidWord) {
				t.Errorf("Feedback(%q, %q) error = %v, want one wrapping ErrInvalidWord", tc.guess, tc.answer, err)
			}
		})
	}
}
