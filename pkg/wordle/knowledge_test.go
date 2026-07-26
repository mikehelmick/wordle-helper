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

// applyAll is a helper for turns that are expected to succeed.
func applyAll(t *testing.T, k *Knowledge, turns ...[2]string) {
	t.Helper()
	for _, turn := range turns {
		if err := k.Apply(turn[0], turn[1]); err != nil {
			t.Fatalf("Apply(%q, %q) returned unexpected error: %v", turn[0], turn[1], err)
		}
	}
}

func TestNewKnowledgeMatchesEverything(t *testing.T) {
	t.Parallel()

	k := NewKnowledge()
	for _, word := range []string{"CRANE", "TOAST", "SASSY", "ABBEY", "ZYMIC"} {
		if !k.Match(word) {
			t.Errorf("Match(%q) = false on fresh Knowledge, want true", word)
		}
	}
}

// TestApplyAccumulatesAcrossTurns is the regression test for the bug that
// prompted this rewrite: green and yellow constraints used to be rebuilt from
// scratch every round, so a letter confirmed in turn one stopped constraining
// anything in turn two.
func TestApplyAccumulatesAcrossTurns(t *testing.T) {
	t.Parallel()

	k := NewKnowledge()
	// Turn one establishes A in the middle. Turn two says nothing about A at
	// all -- under the old reset-per-round behaviour that was enough to forget
	// it.
	applyAll(t, k,
		[2]string{"CRANE", "..G.."},
		[2]string{"MILKY", "....."},
	)

	if !k.Match("TOAST") {
		t.Error("Match(\"TOAST\") = false, want true: it satisfies both turns")
	}
	if k.Match("ABOUT") {
		t.Error("Match(\"ABOUT\") = true, want false: turn one proved position 3 is A")
	}
	if k.Match("TOADY") {
		t.Error("Match(\"TOADY\") = true, want false: turn two ruled out Y")
	}
}

// TestApplyLearnsExactLetterCounts covers the deduction the old exclusion-set
// model could not express. A gray tile for a letter that is also green or
// yellow in the same guess is not "this letter is absent" -- it is "there are
// no more of them than you have already seen".
func TestApplyLearnsExactLetterCounts(t *testing.T) {
	t.Parallel()

	t.Run("gray duplicate caps the count", func(t *testing.T) {
		t.Parallel()
		k := NewKnowledge()
		// ERASE against a word with exactly one E, in the last position.
		applyAll(t, k, [2]string{"ERASE", "..Y.G"})

		if !k.Match("ABIDE") {
			t.Error("Match(\"ABIDE\") = false, want true: one E, ends in E, contains A off position 3")
		}
		if k.Match("AGLEE") {
			t.Error("Match(\"AGLEE\") = true, want false: the gray E proves there is exactly one E")
		}
	})

	t.Run("green duplicates raise the minimum", func(t *testing.T) {
		t.Parallel()
		k := NewKnowledge()
		// Two green Ss and a gray S: exactly two Ss, at positions 3 and 4.
		applyAll(t, k, [2]string{"SASSY", "..GGG"})

		if !k.Match("MOSSY") {
			t.Error("Match(\"MOSSY\") = false, want true")
		}
		if k.Match("PESTS") {
			t.Error("Match(\"PESTS\") = true, want false: S is not at positions 3 and 4")
		}
	})

	t.Run("a letter absent everywhere is excluded", func(t *testing.T) {
		t.Parallel()
		k := NewKnowledge()
		applyAll(t, k, [2]string{"CRANE", "....."})

		for _, word := range []string{"TRACK", "NOISE", "CHOMP"} {
			if k.Match(word) {
				t.Errorf("Match(%q) = true, want false: it reuses a letter shown absent", word)
			}
		}
		if !k.Match("SILTY") {
			t.Error("Match(\"SILTY\") = false, want true: it avoids every absent letter")
		}
	})
}

func TestApplyBansYellowPositions(t *testing.T) {
	t.Parallel()

	k := NewKnowledge()
	applyAll(t, k, [2]string{"CRANE", "..Y.."})

	if k.Match("TOAST") {
		t.Error("Match(\"TOAST\") = true, want false: A cannot stay at position 3")
	}
	if !k.Match("ADOPT") {
		t.Error("Match(\"ADOPT\") = false, want true: it has an A somewhere other than position 3")
	}
	if k.Match("SHOUT") {
		t.Error("Match(\"SHOUT\") = true, want false: it has no A at all")
	}
}

func TestGuessedWordsAreNotSuggested(t *testing.T) {
	t.Parallel()

	k := NewKnowledge()
	applyAll(t, k, [2]string{"ABIDE", "GGGGG"})

	if !k.Guessed("ABIDE") {
		t.Error("Guessed(\"ABIDE\") = false, want true")
	}
	if k.Match("ABIDE") {
		t.Error("Match(\"ABIDE\") = true, want false: it has already been played")
	}
}

func TestApplyRejectsContradictions(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		turns [][2]string
	}{
		{
			name: "two different letters green in one position",
			turns: [][2]string{{"CRANE", "G...."}, {"TOAST", "G...."}},
		},
		{
			name: "a letter is green where it was previously ruled out",
			turns: [][2]string{{"CRANE", ".Y..."}, {"TRACE", ".G..."}},
		},
		{
			name: "a known green is reported misplaced",
			turns: [][2]string{{"CRANE", "G...."}, {"CHOMP", "Y...."}},
		},
		{
			name: "a known green is reported absent",
			turns: [][2]string{{"CRANE", "G...."}, {"CHOMP", "....."}},
		},
		{
			name: "a letter is required and forbidden at once",
			turns: [][2]string{{"CRANE", "..Y.."}, {"ABIDE", "....."}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			k := NewKnowledge()
			last := len(tc.turns) - 1
			applyAll(t, k, tc.turns[:last]...)

			before := k.String()
			err := k.Apply(tc.turns[last][0], tc.turns[last][1])
			if !errors.Is(err, ErrContradiction) {
				t.Fatalf("Apply(%q, %q) error = %v, want one wrapping ErrContradiction",
					tc.turns[last][0], tc.turns[last][1], err)
			}
			// A rejected turn must be a no-op, or the caller cannot recover by
			// simply re-entering it.
			if after := k.String(); after != before {
				t.Errorf("knowledge changed despite the error:\n before: %s\n  after: %s", before, after)
			}
		})
	}
}

func TestApplyRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		guess   string
		pattern string
		wantErr error
	}{
		{"short guess", "CRAN", ".....", ErrInvalidWord},
		{"long guess", "CRANES", ".....", ErrInvalidWord},
		{"digits in guess", "CR4NE", ".....", ErrInvalidWord},
		{"multi-byte rune in guess", "CRÁN", ".....", ErrInvalidWord},
		{"short pattern", "CRANE", "....", ErrInvalidPattern},
		{"long pattern", "CRANE", "......", ErrInvalidPattern},
		{"unknown pattern character", "CRANE", "..X..", ErrInvalidPattern},
		// These used to be handed to regexp.Compile. A character class is
		// exactly five bytes, and a quantifier is a valid pattern length too,
		// so neither was caught by a length check alone.
		{"regex character class as guess", "[A-Z]", ".....", ErrInvalidWord},
		{"regex quantifier as pattern", "CRANE", ".*.*.", ErrInvalidPattern},
		{"regex anchors as pattern", "CRANE", "^...$", ErrInvalidPattern},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			k := NewKnowledge()
			err := k.Apply(tc.guess, tc.pattern)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Apply(%q, %q) error = %v, want one wrapping %v", tc.guess, tc.pattern, err, tc.wantErr)
			}
			if got := k.String(); got != NewKnowledge().String() {
				t.Errorf("invalid input mutated the knowledge: %s", got)
			}
		})
	}
}

func TestApplyNormalizesInput(t *testing.T) {
	t.Parallel()

	upper := NewKnowledge()
	applyAll(t, upper, [2]string{"CRANE", "..G.."})

	lower := NewKnowledge()
	applyAll(t, lower, [2]string{" crane ", " ..g.. "})

	if got, want := lower.String(), upper.String(); got != want {
		t.Errorf("lowercase input produced different knowledge:\n got: %s\nwant: %s", got, want)
	}
}

func TestMatchRejectsMalformedWords(t *testing.T) {
	t.Parallel()

	k := NewKnowledge()
	for _, word := range []string{"", "CRAN", "CRANES", "cr4ne", "CRÁN"} {
		if k.Match(word) {
			t.Errorf("Match(%q) = true, want false", word)
		}
	}
}
