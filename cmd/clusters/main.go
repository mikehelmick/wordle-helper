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

// Command clusters reports groups of words that differ in exactly one
// position, such as _IGHT -> FIGHT LIGHT MIGHT NIGHT.
//
// These are the traps that turn a solved-looking board into a coin flip: once
// the board is down to one such cluster, no amount of filtering distinguishes
// the members, and the next guess has to be chosen to split them.
//
//	go run ./cmd/clusters          # every cluster, largest first
//	go run ./cmd/clusters 8        # only clusters with 8 or more members
package main

import (
	"cmp"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/mikehelmick/wordle-helper/pkg/wordle"
)

// blank marks the position a cluster's members disagree on.
const blank = '_'

func main() {
	minSize := 2
	if len(os.Args) > 1 {
		n, err := strconv.Atoi(os.Args[1])
		if err != nil || n < 2 {
			fmt.Fprintf(os.Stderr, "clusters: minimum size must be an integer >= 2, got %q\n", os.Args[1])
			os.Exit(1)
		}
		minSize = n
	}
	if err := run(minSize); err != nil {
		fmt.Fprintf(os.Stderr, "clusters: %v\n", err)
		os.Exit(1)
	}
}

func run(minSize int) error {
	dict, err := wordle.DefaultDictionary()
	if err != nil {
		return err
	}

	// Every word contributes one pattern per position; words sharing a pattern
	// differ only there.
	clusters := make(map[string][]string)
	for _, word := range dict.Words {
		for _, p := range patterns(word) {
			clusters[p] = append(clusters[p], word)
		}
	}

	type cluster struct {
		pattern string
		members []string
	}
	found := make([]cluster, 0, len(clusters))
	for pattern, members := range clusters {
		if len(members) >= minSize {
			found = append(found, cluster{pattern, members})
		}
	}

	// Largest first, then alphabetically, so the output is stable across runs
	// despite the random map iteration order that built it.
	slices.SortFunc(found, func(a, b cluster) int {
		if c := cmp.Compare(len(b.members), len(a.members)); c != 0 {
			return c
		}
		return cmp.Compare(a.pattern, b.pattern)
	})

	for _, c := range found {
		fmt.Printf("%s (%d)  %s\n", c.pattern, len(c.members), strings.Join(c.members, " "))
	}
	fmt.Printf("\n%d clusters of %d or more, from %d words\n", len(found), minSize, len(dict.Words))
	return nil
}

// patterns returns word with each position blanked out in turn, so CRANE gives
// _RANE, C_ANE, CR_NE, CRA_E and CRAN_.
func patterns(word string) []string {
	out := make([]string, wordle.WordLen)
	buf := []byte(word)
	for i := range wordle.WordLen {
		orig := buf[i]
		buf[i] = blank
		out[i] = string(buf)
		buf[i] = orig
	}
	return out
}
