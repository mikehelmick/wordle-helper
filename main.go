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

package main

import (
	"fmt"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/mikehelmick/wordle-helper/pkg/wordle"
)

const (
	notInWord = '.'
	wrongSpot = 'Y'
	rightSpot = 'G'
)

func validator() func(string) error {
	return func(s string) error {
		s = strings.ToUpper(s)
		if s == "EXIT" {
			return nil
		}
		if len(s) != 5 {
			return fmt.Errorf("incorrect length")
		}
		return nil
	}
}

func guessValidator() func(string) error {
	return func(s string) error {
		if len(s) != 5 {
			return fmt.Errorf("incorrect length")
		}
		s = strings.ToUpper(s)
		for _, r := range s {
			if r != notInWord && r != wrongSpot && r != rightSpot {
				return fmt.Errorf("invalid characters")
			}
		}
		return nil
	}
}

func main() {
	fmt.Printf("Wordle helper - I'm only interested in wrong guesses\nType 'EXIT' to quit\n")
	k := wordle.NewKnowledge()
	for i := 1; i <= 5; i++ {
		prompt := promptui.Prompt{
			Label:    fmt.Sprintf("Invalid guess %d", i),
			Validate: validator(),
		}

		guess, err := prompt.Run()
		if err != nil {
			panic(err)
		}
		guess = strings.ToUpper(guess)
		if guess == "EXIT" {
			return
		}

		prompt = promptui.Prompt{
			Label:    "Enter pattern . = not in word, Y = wrong spot, G = correct: ",
			Validate: guessValidator(),
		}
		pattern, err := prompt.Run()
		if err != nil {
			panic(err)
		}
		pattern = strings.ToUpper(pattern)

		k.PreviousWords[guess] = struct{}{}
		// Update knowledge
		k.Contains = make([]string, 0)
		k.Exact = ""
		for i, r := range pattern {
			l := string(guess[i])
			if r == wrongSpot {
				k.AddNotInPosition(l, i)
				k.Contains = append(k.Contains, l)
				k.Exact = k.Exact + "."
				continue
			}
			if r == rightSpot {
				k.Exact = k.Exact + l
				continue
			}
			if r == notInWord {
				k.Exclude = fmt.Sprintf("%s%s", k.Exclude, l)
				k.Exact = k.Exact + "."
				continue
			}
		}
		k.CleanExclude()

		//fmt.Printf("KNOWLEDGE: %+v\n", *k)

		possible, err := wordle.Suggest(k)
		if err != nil {
			panic(fmt.Sprintf("unable to suggest: %v", err))
		}
		fmt.Printf("Found %d suggestions\n", len(possible))
		display := true
		if l := len(possible); l > 100 {
			prompt := promptui.Prompt{
				Label:     fmt.Sprintf("Display all %d suggestions?", l),
				IsConfirm: true,
			}

			_, err := prompt.Run()
			if err != nil {
				display = false
			}
		}
		if display {
			output := strings.Join(possible, " ")
			fmt.Printf("%s\n", output)
		}
	}
}
