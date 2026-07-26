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
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// The solver exists twice: here, and as a TypeScript port behind the web UI.
// Two implementations of one algorithm drift, and the drift is invisible until
// somebody gets a wrong answer. testdata/vectors.json is the shared contract
// that stops it: this test generates the expected values from the Go code, and
// the TypeScript suite asserts against the very same file.
//
// Regenerate after any deliberate behaviour change:
//
//	go test ./pkg/wordle -run TestVectors -update
//
// Then run the TypeScript tests. If they fail, the two implementations disagree
// and one of them is wrong.

var update = flag.Bool("update", false, "regenerate testdata/vectors.json")

const vectorsPath = "testdata/vectors.json"

type vectorFile struct {
	Note     string         `json:"note"`
	WordLen  int            `json:"wordLen"`
	Feedback []feedbackCase `json:"feedback"`
	Solve    []solveCase    `json:"solve"`
}

type feedbackCase struct {
	Name   string `json:"name"`
	Guess  string `json:"guess"`
	Answer string `json:"answer"`
	// Pattern is generated output, not input.
	Pattern string `json:"pattern"`
}

type turn struct {
	Guess   string `json:"guess"`
	Pattern string `json:"pattern"`
}

type solveCase struct {
	Name string `json:"name"`
	// Dictionary is omitted when the case runs against the full embedded word
	// list, which both implementations ship identically.
	Dictionary []string `json:"dictionary,omitempty"`
	Turns      []turn   `json:"turns"`
	// Expected is generated output, not input.
	Expected []string `json:"expected"`
}

// vectorFixture is the small dictionary the fixture-based cases run against.
var vectorFixture = []string{
	"ABBEY", "ABIDE", "AGLEE", "ALLEY", "CRANE",
	"LEVEL", "MOSSY", "SASSY", "SPEED", "TOAST",
}

// feedbackInputs are the (guess, answer) pairs whose patterns get generated.
var feedbackInputs = []feedbackCase{
	{Name: "no letters in common", Guess: "CHIRP", Answer: "SLUMS"},
	{Name: "exact match", Guess: "CRANE", Answer: "CRANE"},
	{Name: "one green", Guess: "CRANE", Answer: "TOAST"},
	{Name: "one yellow", Guess: "CRANE", Answer: "SASSY"},
	{Name: "duplicate in guess, single in answer", Guess: "SPEED", Answer: "ABIDE"},
	{Name: "duplicate in guess, duplicate in answer", Guess: "SPEED", Answer: "ERASE"},
	{Name: "greens consume the available duplicates", Guess: "SASSY", Answer: "MOSSY"},
	{Name: "duplicate split across yellow and green", Guess: "ALLEY", Answer: "LEVEL"},
	{Name: "triple letter in guess", Guess: "MUMMY", Answer: "MOTTO"},
	{Name: "all letters present but misplaced", Guess: "SPEED", Answer: "PSEUD"},
}

// solveInputs are the turn sequences whose surviving candidates get generated.
var solveInputs = []solveCase{
	{
		Name:       "no knowledge matches the whole dictionary",
		Dictionary: vectorFixture,
	},
	{
		Name:       "a green fixes a position",
		Dictionary: vectorFixture,
		Turns:      []turn{{"CRANE", "..G.."}},
	},
	{
		Name:       "greens survive a later turn that does not mention them",
		Dictionary: vectorFixture,
		Turns:      []turn{{"CRANE", "..G.."}, {"MILKY", "....."}},
	},
	{
		Name:       "a gray duplicate caps the letter count",
		Dictionary: vectorFixture,
		Turns:      []turn{{"ERASE", "..Y.G"}},
	},
	{
		Name:       "a yellow requires the letter somewhere else",
		Dictionary: vectorFixture,
		Turns:      []turn{{"SPEED", "Y...."}},
	},
	{
		Name:       "a played word is never suggested again",
		Dictionary: vectorFixture,
		Turns:      []turn{{"ALLEY", "GGGGG"}},
	},
	{
		Name:  "three turns against the full dictionary",
		Turns: []turn{{"CRANE", "..G.."}, {"ABAFT", "..G.G"}, {"GHAST", "..GGG"}},
	},
	{
		Name:  "repeated letters against the full dictionary",
		Turns: []turn{{"SASSY", "..GGG"}},
	},
	{
		Name:  "exactly one E against the full dictionary",
		Turns: []turn{{"ERASE", "..Y.G"}, {"BUILD", "....."}},
	},
}

func TestVectors(t *testing.T) {
	generated, err := generateVectors()
	if err != nil {
		t.Fatalf("generating vectors: %v", err)
	}

	if *update {
		if err := writeVectors(generated); err != nil {
			t.Fatalf("writing %s: %v", vectorsPath, err)
		}
		t.Logf("regenerated %s (%d feedback cases, %d solve cases)",
			vectorsPath, len(generated.Feedback), len(generated.Solve))
		return
	}

	data, err := os.ReadFile(vectorsPath)
	if err != nil {
		t.Fatalf("reading %s (run: go test ./pkg/wordle -run TestVectors -update): %v", vectorsPath, err)
	}
	var stored vectorFile
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatalf("parsing %s: %v", vectorsPath, err)
	}

	if stored.WordLen != WordLen {
		t.Errorf("wordLen = %d, want %d", stored.WordLen, WordLen)
	}

	if got, want := len(stored.Feedback), len(generated.Feedback); got != want {
		t.Fatalf("stored file has %d feedback cases, code defines %d; regenerate with -update", got, want)
	}
	for i, want := range generated.Feedback {
		got := stored.Feedback[i]
		if got.Guess != want.Guess || got.Answer != want.Answer {
			t.Errorf("feedback[%d] inputs = (%q, %q), want (%q, %q); regenerate with -update",
				i, got.Guess, got.Answer, want.Guess, want.Answer)
			continue
		}
		if got.Pattern != want.Pattern {
			t.Errorf("feedback[%d] %s: stored pattern %q, implementation produces %q",
				i, want.Name, got.Pattern, want.Pattern)
		}
	}

	if got, want := len(stored.Solve), len(generated.Solve); got != want {
		t.Fatalf("stored file has %d solve cases, code defines %d; regenerate with -update", got, want)
	}
	for i, want := range generated.Solve {
		got := stored.Solve[i]
		if got.Name != want.Name {
			t.Errorf("solve[%d] name = %q, want %q; regenerate with -update", i, got.Name, want.Name)
			continue
		}
		if !slices.Equal(got.Expected, want.Expected) {
			t.Errorf("solve[%d] %s: stored %d candidates, implementation produces %d\n stored: %v\n   got: %v",
				i, want.Name, len(got.Expected), len(want.Expected), got.Expected, want.Expected)
		}
	}
}

func generateVectors() (*vectorFile, error) {
	out := &vectorFile{
		Note: "Generated by `go test ./pkg/wordle -run TestVectors -update`. " +
			"Shared contract between the Go solver and the TypeScript port; do not edit by hand. " +
			"A solve case with no `dictionary` runs against the full embedded word list.",
		WordLen:  WordLen,
		Feedback: make([]feedbackCase, 0, len(feedbackInputs)),
		Solve:    make([]solveCase, 0, len(solveInputs)),
	}

	for _, in := range feedbackInputs {
		pattern, err := Feedback(in.Guess, in.Answer)
		if err != nil {
			return nil, err
		}
		in.Pattern = pattern
		out.Feedback = append(out.Feedback, in)
	}

	embeddedDict, err := DefaultDictionary()
	if err != nil {
		return nil, err
	}

	for _, in := range solveInputs {
		dict := embeddedDict
		if in.Dictionary != nil {
			if dict, err = NewDictionary(in.Dictionary); err != nil {
				return nil, err
			}
		}

		k := NewKnowledge()
		for _, tn := range in.Turns {
			if err := k.Apply(tn.Guess, tn.Pattern); err != nil {
				return nil, err
			}
		}
		expected, err := Suggest(dict, k)
		if err != nil {
			return nil, err
		}

		in.Expected = expected
		if in.Turns == nil {
			in.Turns = []turn{}
		}
		out.Solve = append(out.Solve, in)
	}

	return out, nil
}

func writeVectors(v *vectorFile) error {
	if err := os.MkdirAll(filepath.Dir(vectorsPath), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(vectorsPath, append(data, '\n'), 0o644)
}
