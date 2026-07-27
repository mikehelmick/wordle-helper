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

// Command tester plays a whole game against a known answer, guessing the first
// word the solver still considers possible, and reports how the candidate set
// collapses.
//
//	go run ./cmd/tester           # uses the default answer
//	go run ./cmd/tester TOAST     # solves for a specific word
//
// The real coverage lives in pkg/wordle's tests; this is for eyeballing
// behaviour by hand. It replaces a version that hand-built a Knowledge from
// exported fields, printed any error and then carried on with a nil result.
package main

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/mikehelmick/wordle-helper/pkg/wordle"
)

// defaultAnswer has a repeated letter, which exercises the trickiest part of
// the feedback rules.
const defaultAnswer = "TOAST"

// opener is the first guess; it has no repeats and covers three vowels.
const opener = "CRANE"

const maxGuesses = 6

func main() {
	answer := defaultAnswer
	if len(os.Args) > 1 {
		answer = os.Args[1]
	}
	if err := run(strings.ToUpper(strings.TrimSpace(answer))); err != nil {
		fmt.Fprintf(os.Stderr, "tester: %v\n", err)
		os.Exit(1)
	}
}

func run(answer string) error {
	dict := wordle.DefaultDictionary()
	if err := wordle.ValidWord(answer); err != nil {
		return err
	}
	if !slices.Contains(dict.Words, answer) {
		return fmt.Errorf("%q is not in the dictionary", answer)
	}

	k := wordle.NewKnowledge()
	guess := opener
	for round := 1; round <= maxGuesses; round++ {
		pattern, err := wordle.Feedback(guess, answer)
		if err != nil {
			return err
		}
		fmt.Printf("%d. %s  %s\n", round, guess, pattern)
		if guess == answer {
			fmt.Printf("   solved in %d\n", round)
			return nil
		}

		if err := k.Apply(guess, pattern); err != nil {
			return fmt.Errorf("applying %s/%s: %w", guess, pattern, err)
		}
		possible, err := wordle.Suggest(dict, k)
		if err != nil {
			return err
		}
		fmt.Printf("   %d remaining  %s\n", len(possible), preview(possible, 12))
		if len(possible) == 0 {
			return fmt.Errorf("no candidate remains for %q", answer)
		}
		guess = possible[0]
	}
	return fmt.Errorf("did not solve %q within %d guesses", answer, maxGuesses)
}

func preview(words []string, n int) string {
	if len(words) <= n {
		return strings.Join(words, " ")
	}
	return strings.Join(words[:n], " ") + " ..."
}
