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
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

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
	if err := run(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "wordle-helper: %v\n", err)
		os.Exit(1)
	}
}

// prompter reads answers from a stream, re-asking until one validates.
//
// This replaces github.com/manifoldco/promptui, which was the module's only
// dependency and had not seen a release since 2021. Reading lines is all the
// tool ever needed, so the two features it used are cheaper to own than to
// depend on -- and taking an io.Reader makes the whole session testable, which
// a terminal-control library is not.
type prompter struct {
	in  *bufio.Scanner
	out io.Writer
}

func newPrompter(in io.Reader, out io.Writer) *prompter {
	return &prompter{in: bufio.NewScanner(in), out: out}
}

// ask writes label, reads a line, and returns it once validate accepts it.
// Invalid answers are reported and re-asked. End of input returns errQuit.
func (p *prompter) ask(label string, validate func(string) error) (string, error) {
	for {
		fmt.Fprintf(p.out, "%s: ", label)

		if !p.in.Scan() {
			if err := p.in.Err(); err != nil {
				return "", err
			}
			fmt.Fprintln(p.out)
			return "", errQuit
		}

		answer := strings.TrimSpace(p.in.Text())
		if err := validate(answer); err != nil {
			fmt.Fprintf(p.out, "  %v\n", err)
			continue
		}
		return answer, nil
	}
}

// confirm asks a yes/no question, defaulting to no.
func (p *prompter) confirm(label string) (bool, error) {
	fmt.Fprintf(p.out, "%s [y/N]: ", label)

	if !p.in.Scan() {
		if err := p.in.Err(); err != nil {
			return false, err
		}
		fmt.Fprintln(p.out)
		return false, errQuit
	}

	switch strings.ToLower(strings.TrimSpace(p.in.Text())) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}

func run(stdin io.Reader, stdout io.Writer) error {
	dict := wordle.DefaultDictionary()

	p := newPrompter(stdin, stdout)
	fmt.Fprintf(stdout, "Wordle helper - I'm only interested in wrong guesses\nType %q to quit\n\n", exitWord)

	k := wordle.NewKnowledge()
	// round advances only on a successfully recorded turn, so mistyped feedback
	// costs a retry rather than one of the five rounds.
	for round := 1; round <= maxRounds; {
		guess, err := p.ask(fmt.Sprintf("Wrong guess %d", round), validateGuess)
		if err != nil {
			return orQuit(err)
		}
		guess = strings.ToUpper(guess)
		if guess == exitWord {
			return nil
		}

		pattern, err := p.ask(
			fmt.Sprintf("Pattern (%c = not in word, %c = wrong spot, %c = correct)",
				wordle.Absent, wordle.Present, wordle.Correct),
			validatePattern,
		)
		if err != nil {
			return orQuit(err)
		}

		if err := k.Apply(guess, pattern); err != nil {
			// Feedback that cannot be reconciled with earlier rounds is a typo,
			// not a fatal condition. Apply left the knowledge untouched, so the
			// round can simply be entered again.
			fmt.Fprintf(stdout, "  %v\n  Nothing was recorded; please re-enter this round.\n\n", err)
			continue
		}

		possible, err := wordle.Suggest(dict, k)
		if err != nil {
			return err
		}
		if err := report(p, stdout, dict, possible); err != nil {
			return orQuit(err)
		}
		round++
	}
	return nil
}

// validateGuess accepts the exit word or exactly five ASCII letters.
//
// It defers to wordle.ValidWord rather than a bare length check: a five-*byte*
// string can hold fewer than five runes, and indexing one by byte splits a
// multi-byte rune into nonsense that then flows straight into the solver's
// constraints.
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

// orQuit turns a quit signal into a successful return and leaves real failures
// alone.
func orQuit(err error) error {
	if errors.Is(err, errQuit) {
		return nil
	}
	return err
}

func report(p *prompter, out io.Writer, dict *wordle.Dictionary, possible []string) error {
	if len(possible) == 0 {
		fmt.Fprintf(out, "Found 0 suggestions\n")
		fmt.Fprintf(out, "No word matches this feedback - check the patterns entered so far.\n\n")
		return nil
	}

	// Most legal guesses have never been answers, so the raw count overstates
	// how much is really left. Reporting both keeps the tool honest -- the
	// unlikely words are still listed, just not led with.
	likely := dict.Answers(possible)
	fmt.Fprintf(out, "Found %d suggestions", len(possible))
	switch {
	case len(likely) == len(possible):
		// Every survivor is plausible; the distinction is not worth drawing.
	case len(likely) == 1:
		fmt.Fprintf(out, ", 1 of which has been an answer before")
	default:
		fmt.Fprintf(out, ", %d of which have been answers before", len(likely))
	}
	fmt.Fprintln(out)

	if len(likely) > 0 && len(likely) <= listThreshold {
		fmt.Fprintf(out, "Likely: %s\n", strings.Join(likely, " "))
	}

	if len(possible) > listThreshold {
		show, err := p.confirm(fmt.Sprintf("Display all %d suggestions", len(possible)))
		if err != nil {
			return err
		}
		if !show {
			fmt.Fprintln(out)
			return nil
		}
	}

	fmt.Fprintf(out, "%s\n\n", strings.Join(possible, " "))
	return nil
}
