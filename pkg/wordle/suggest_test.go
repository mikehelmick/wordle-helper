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
	"slices"
	"testing"
)

// fixture is a deliberately small dictionary, so expectations can be written
// out in full rather than counted.
func fixture(t *testing.T) *Dictionary {
	t.Helper()
	d, err := NewDictionary([]string{
		"ABBEY", "ABIDE", "AGLEE", "ALLEY", "CRANE",
		"LEVEL", "MOSSY", "SASSY", "SPEED", "TOAST",
	})
	if err != nil {
		t.Fatalf("building fixture dictionary: %v", err)
	}
	return d
}

// TestSuggestWithNoKnowledge is the regression test for a solver that returned
// nothing at all when asked before any guess had been entered. The unset green
// mask was compiled as an empty regular expression, whose empty match was then
// read as "no match" -- so every word was skipped.
func TestSuggestWithNoKnowledge(t *testing.T) {
	t.Parallel()

	dict := fixture(t)
	got, err := Suggest(dict, NewKnowledge())
	if err != nil {
		t.Fatalf("Suggest returned unexpected error: %v", err)
	}
	if !slices.Equal(got, dict.Words) {
		t.Errorf("Suggest = %v, want the whole dictionary %v", got, dict.Words)
	}
}

func TestSuggest(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		turns [][2]string
		want  []string
	}{
		{
			name:  "green fixes a position",
			turns: [][2]string{{"CRANE", "..G.."}},
			want:  []string{"TOAST"},
		},
		{
			name: "greens accumulate across turns",
			// The second turn says nothing about A. Only because turn one is
			// still in force does the A stay pinned to position 3.
			turns: [][2]string{{"CRANE", "..G.."}, {"MILKY", "....."}},
			want:  []string{"TOAST"},
		},
		{
			name: "a gray duplicate caps the letter count",
			// One E, in last position. AGLEE has two, so it is out even though
			// it satisfies every positional constraint.
			turns: [][2]string{{"ERASE", "..Y.G"}},
			want:  []string{"ABIDE"},
		},
		{
			name: "yellow requires the letter elsewhere",
			// SASSY is excluded too: a yellow S in position 1 means the answer
			// has an S, but not there.
			turns: [][2]string{{"SPEED", "Y...."}},
			want:  []string{"MOSSY", "TOAST"},
		},
		{
			name:  "previous guesses are never suggested",
			turns: [][2]string{{"ALLEY", "GGGGG"}},
			want:  []string{},
		},
		{
			name:  "exhaustive exclusions leave nothing",
			turns: [][2]string{{"CRANE", "....."}, {"TOAST", "....."}},
			want:  []string{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			k := NewKnowledge()
			applyAll(t, k, tc.turns...)

			got, err := Suggest(fixture(t), k)
			if err != nil {
				t.Fatalf("Suggest returned unexpected error: %v", err)
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("Suggest = %v, want %v\nknowledge: %v", got, tc.want, k)
			}
		})
	}
}

func TestSuggestReturnsDictionaryOrder(t *testing.T) {
	t.Parallel()

	got, err := Suggest(fixture(t), NewKnowledge())
	if err != nil {
		t.Fatalf("Suggest returned unexpected error: %v", err)
	}
	if !slices.IsSorted(got) {
		t.Errorf("Suggest = %v, want sorted output", got)
	}
}

func TestSuggestRejectsNilArguments(t *testing.T) {
	t.Parallel()

	if _, err := Suggest(nil, NewKnowledge()); err == nil {
		t.Error("Suggest(nil, k) = nil error, want a failure")
	}
	if _, err := Suggest(fixture(t), nil); err == nil {
		t.Error("Suggest(d, nil) = nil error, want a failure")
	}
}

// TestSuggestAlwaysRetainsTheAnswer plays real games end to end against the
// embedded dictionary. Whatever else the filter does, it must never discard the
// word actually being sought.
func TestSuggestAlwaysRetainsTheAnswer(t *testing.T) {
	t.Parallel()

	dict, err := DefaultDictionary()
	if err != nil {
		t.Fatalf("DefaultDictionary() returned unexpected error: %v", err)
	}

	// Answers chosen for their duplicate letters, which is where the filtering
	// rules are hardest to get right.
	answers := []string{"TOAST", "SASSY", "ABBEY", "LEVEL", "SPEED", "MUMMY", "EERIE", "CRANE"}

	for _, answer := range answers {
		t.Run(answer, func(t *testing.T) {
			t.Parallel()
			k := NewKnowledge()
			guess := "CRANE"
			for round := 1; round <= 8; round++ {
				if guess == answer {
					return // Solved.
				}
				pattern, err := Feedback(guess, answer)
				if err != nil {
					t.Fatalf("Feedback(%q, %q): %v", guess, answer, err)
				}
				if err := k.Apply(guess, pattern); err != nil {
					t.Fatalf("Apply(%q, %q): %v", guess, pattern, err)
				}

				possible, err := Suggest(dict, k)
				if err != nil {
					t.Fatalf("Suggest: %v", err)
				}
				if !slices.Contains(possible, answer) {
					t.Fatalf("answer %q was filtered out after guessing %q (pattern %q); knowledge: %v",
						answer, guess, pattern, k)
				}
				guess = possible[0]
			}
			// Convergence is not the property under test -- guessing the
			// alphabetically first candidate is a deliberately poor strategy,
			// and clusters like _A__Y defeat it. Retaining the answer is.
			t.Logf("did not converge on %q within 8 rounds, which this strategy need not", answer)
		})
	}
}

func BenchmarkSuggest(b *testing.B) {
	dict, err := DefaultDictionary()
	if err != nil {
		b.Fatalf("DefaultDictionary() returned unexpected error: %v", err)
	}
	k := NewKnowledge()
	if err := k.Apply("CRANE", "..G.."); err != nil {
		b.Fatalf("Apply returned unexpected error: %v", err)
	}

	b.ReportAllocs()
	for b.Loop() {
		if _, err := Suggest(dict, k); err != nil {
			b.Fatal(err)
		}
	}
}
