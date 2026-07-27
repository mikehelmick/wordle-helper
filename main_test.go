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
	"strings"
	"testing"
)

// session runs a whole CLI session against scripted input and returns what the
// user would have seen. Dropping promptui for plain reads is what makes this
// possible; a terminal-control library cannot be driven from a string.
func session(t *testing.T, input string) string {
	t.Helper()
	var out strings.Builder
	if err := run(strings.NewReader(input), &out); err != nil {
		t.Fatalf("run returned unexpected error: %v\noutput so far:\n%s", err, out.String())
	}
	return out.String()
}

func requireContains(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Errorf("output does not contain %q\n--- full output ---\n%s", want, got)
	}
}

func requireNotContains(t *testing.T, got, want string) {
	t.Helper()
	if strings.Contains(got, want) {
		t.Errorf("output unexpectedly contains %q\n--- full output ---\n%s", want, got)
	}
}

func TestSolvesAcrossTurns(t *testing.T) {
	t.Parallel()

	got := session(t, strings.Join([]string{
		"CRANE", "..G..", "n", // 323 suggestions, decline the listing
		"ABAFT", "..G.G", // 15, printed without asking
		"GHAST", "..GGG", // 3
		"EXIT",
	}, "\n")+"\n")

	requireContains(t, got, "Found 323 suggestions")
	requireContains(t, got, "Found 15 suggestions")
	requireContains(t, got, "Found 3 suggestions")
	requireContains(t, got, "LOAST PLAST TOAST")
}

func TestDecliningTheListingSuppressesIt(t *testing.T) {
	t.Parallel()

	declined := session(t, "CRANE\n..G..\nn\nEXIT\n")
	requireContains(t, declined, "Display all 323 suggestions")
	requireNotContains(t, declined, "ABAFT")

	accepted := session(t, "CRANE\n..G..\ny\nEXIT\n")
	requireContains(t, accepted, "ABAFT")
}

func TestInvalidInputIsRejectedAndReasked(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name, input, wantMessage string
	}{
		{"short guess", "CRAN\nCRANE\n..G..\nn\nEXIT\n", "invalid word"},
		{"digits in guess", "CR4NE\nCRANE\n..G..\nn\nEXIT\n", "invalid word"},
		// Five bytes but four runes. A length check alone would let this
		// through and corrupt the constraints.
		{"multi-byte rune", "CRÁN\nCRANE\n..G..\nn\nEXIT\n", "invalid word"},
		// Both of these used to reach regexp.Compile.
		{"regex class as guess", "[A-Z]\nCRANE\n..G..\nn\nEXIT\n", "invalid word"},
		{"regex quantifier as pattern", "CRANE\n.*.*.\n..G..\nn\nEXIT\n", "invalid pattern"},
		{"bad pattern character", "CRANE\n..X..\n..G..\nn\nEXIT\n", "invalid pattern"},
		{"short pattern", "CRANE\n..G.\n..G..\nn\nEXIT\n", "invalid pattern"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := session(t, tc.input)
			requireContains(t, got, tc.wantMessage)
			// Rejected input must not cost a round.
			requireContains(t, got, "Found 323 suggestions")
			requireContains(t, got, "Wrong guess 2")
		})
	}
}

func TestContradictionReplaysTheRound(t *testing.T) {
	t.Parallel()

	// C is green in round one, then absent in round two: impossible.
	got := session(t, strings.Join([]string{
		"CRANE", "G....", "n",
		"CHOMP", ".....", // contradicts the green C
		"EXIT",
	}, "\n")+"\n")

	requireContains(t, got, "already known to be at position 1")
	requireContains(t, got, "Nothing was recorded")
	// The round is re-asked rather than consumed.
	requireContains(t, got, "Wrong guess 2")
	requireNotContains(t, got, "Wrong guess 3")
}

func TestExitQuitsImmediately(t *testing.T) {
	t.Parallel()

	got := session(t, "EXIT\n")
	requireContains(t, got, "Wrong guess 1")
	requireNotContains(t, got, "Found")
}

func TestEndOfInputQuitsCleanly(t *testing.T) {
	t.Parallel()

	// No trailing newline and no EXIT: the reader simply runs dry, which is
	// what Ctrl-D does. It must not be treated as a failure.
	if got := session(t, "CRANE\n..G..\nn\n"); !strings.Contains(got, "Found 323 suggestions") {
		t.Errorf("expected the completed round before end of input\n%s", got)
	}
	session(t, "")
	session(t, "CRANE\n")
}

func TestLowercaseInputIsAccepted(t *testing.T) {
	t.Parallel()

	requireContains(t, session(t, "crane\n..g..\nn\nEXIT\n"), "Found 323 suggestions")
}

func TestStopsAfterFiveRounds(t *testing.T) {
	t.Parallel()

	// Five all-grey rounds, each narrowing nothing into a contradiction, would
	// exhaust the loop. Use distinct guesses so nothing is rejected.
	got := session(t, strings.Join([]string{
		"BUXOM", ".....", "n",
		"FRIZZ", ".....", "n",
		"PLYNG", ".....", "n",
		"WHACK", ".....", "n",
		"VEXED", ".....", "n",
	}, "\n")+"\n")

	requireContains(t, got, "Wrong guess 5")
	requireNotContains(t, got, "Wrong guess 6")
}
