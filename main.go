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

// Command wordle-helper narrows down the possible answers to a Wordle puzzle
// from the feedback your wrong guesses have produced.
package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/mikehelmick/wordle-helper/pkg/wordle"
)

const (
	// maxRounds is how many wrong guesses Wordle leaves room for before the
	// final attempt has to be the answer.
	maxRounds = 5

	// listThreshold is the number of suggestions past which the full list is
	// printed only on request.
	listThreshold = 100

	exitWord = "EXIT"
)

// errQuit signals a clean, user-requested shutdown rather than a failure.
var errQuit = errors.New("quit")

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "wordle-helper: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	dict, err := wordle.DefaultDictionary()
	if err != nil {
		return err
	}

	fmt.Printf("Wordle helper - I'm only interested in wrong guesses\nType %q to quit\n", exitWord)

	k := wordle.NewKnowledge()
	// round advances only on a successfully recorded turn, so mistyped
	// feedback costs a retry rather than one of the five rounds.
	for round := 1; round <= maxRounds; {
		guess, err := ask(promptui.Prompt{
			Label:    fmt.Sprintf("Wrong guess %d", round),
			Validate: validateGuess,
		})
		if err != nil {
			return orQuit(err)
		}
		guess = strings.ToUpper(strings.TrimSpace(guess))
		if guess == exitWord {
			return nil
		}

		pattern, err := ask(promptui.Prompt{
			Label: fmt.Sprintf("Pattern (%c = not in word, %c = wrong spot, %c = correct)",
				wordle.Absent, wordle.Present, wordle.Correct),
			Validate: validatePattern,
		})
		if err != nil {
			return orQuit(err)
		}

		if err := k.Apply(guess, pattern); err != nil {
			// Feedback that cannot be reconciled with earlier rounds is a typo,
			// not a fatal condition. Apply left the knowledge untouched, so the
			// round can simply be entered again.
			fmt.Fprintf(os.Stderr, "  %v\n  Nothing was recorded; please re-enter this round.\n", err)
			continue
		}

		possible, err := wordle.Suggest(dict, k)
		if err != nil {
			return err
		}
		if err := report(possible); err != nil {
			return orQuit(err)
		}
		round++
	}
	return nil
}

// validateGuess accepts the exit word or exactly five ASCII letters.
//
// It defers to wordle.ValidWord rather than the bare length check this used to
// do: a five-*byte* string can hold fewer than five runes, and indexing one by
// byte splits a multi-byte rune into nonsense that then flows straight into the
// solver's constraints.
func validateGuess(s string) error {
	s = strings.ToUpper(strings.TrimSpace(s))
	if s == exitWord {
		return nil
	}
	return wordle.ValidWord(s)
}

func validatePattern(s string) error {
	return wordle.ValidPattern(strings.ToUpper(strings.TrimSpace(s)))
}

// ask runs p, reporting Ctrl-C and EOF as errQuit. promptui surfaces both as
// ordinary errors, which the previous version passed to panic -- so quitting
// the program printed a stack trace.
func ask(p promptui.Prompt) (string, error) {
	s, err := p.Run()
	switch {
	case err == nil:
		return s, nil
	case errors.Is(err, promptui.ErrInterrupt), errors.Is(err, promptui.ErrEOF):
		return "", errQuit
	default:
		return "", err
	}
}

// orQuit turns a quit signal into a successful return and leaves real failures
// alone.
func orQuit(err error) error {
	if errors.Is(err, errQuit) {
		return nil
	}
	return err
}

func report(possible []string) error {
	fmt.Printf("Found %d suggestions\n", len(possible))
	if len(possible) == 0 {
		fmt.Println("No word matches this feedback - check the patterns entered so far.")
		return nil
	}

	if len(possible) > listThreshold {
		confirm := promptui.Prompt{
			Label:     fmt.Sprintf("Display all %d suggestions", len(possible)),
			IsConfirm: true,
		}
		_, err := confirm.Run()
		switch {
		case err == nil:
			// Confirmed; fall through and print.
		case errors.Is(err, promptui.ErrInterrupt), errors.Is(err, promptui.ErrEOF):
			return errQuit
		default:
			// Declined, or the prompt failed. Either way, skip the listing.
			return nil
		}
	}

	fmt.Println(strings.Join(possible, " "))
	return nil
}
