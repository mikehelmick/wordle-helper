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
	"fmt"
	"strings"
)

// Feedback returns the pattern Wordle would show for guess against answer.
//
// The CLI never needs this -- the player reads the colors off the screen and
// types them in -- but generating feedback rather than hand-writing it makes
// test cases far cheaper to author, and it is the shared definition the
// TypeScript port is checked against.
//
// The two passes matter for repeated letters. Exact matches claim their letter
// first, and only the leftovers can be reported as misplaced. So guessing SPEED
// against ABIDE marks the first E misplaced and the second absent: ABIDE holds
// just one E, and the first of the guess's two Es consumes it.
func Feedback(guess, answer string) (string, error) {
	guess = strings.ToUpper(strings.TrimSpace(guess))
	answer = strings.ToUpper(strings.TrimSpace(answer))
	if err := ValidWord(guess); err != nil {
		return "", fmt.Errorf("guess: %w", err)
	}
	if err := ValidWord(answer); err != nil {
		return "", fmt.Errorf("answer: %w", err)
	}

	var out [WordLen]byte
	// unclaimed counts the answer's letters that no green tile has taken.
	var unclaimed [alphabetSize]uint8

	for i := range WordLen {
		if guess[i] == answer[i] {
			out[i] = Correct
			continue
		}
		out[i] = Absent
		unclaimed[answer[i]-'A']++
	}

	for i := range WordLen {
		if out[i] == Correct {
			continue
		}
		if l := guess[i] - 'A'; unclaimed[l] > 0 {
			out[i] = Present
			unclaimed[l]--
		}
	}

	return string(out[:]), nil
}
