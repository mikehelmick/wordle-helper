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
	"regexp"
	"strings"
)

var dict = NewDictionary()

type Knowledge struct {
	Exclude       string
	NotInPosition []string
	Contains      []string
	Exact         string
	PreviousWords map[string]struct{}
}

func NewKnowledge() *Knowledge {
	return &Knowledge{
		Exclude:       "",
		NotInPosition: make([]string, 0),
		Contains:      make([]string, 0),
		Exact:         "",
		PreviousWords: make(map[string]struct{}),
	}
}

func (k *Knowledge) AddNotInPosition(s string, pos int) {
	excludeExp := ""
	for i := 0; i < 5; i++ {
		if i == pos {
			excludeExp = excludeExp + s
		} else {
			excludeExp = excludeExp + "."
		}
	}
	k.NotInPosition = append(k.NotInPosition, excludeExp)
}

func (k *Knowledge) CleanExclude() {
	for _, c := range k.Exact {
		k.Exclude = strings.ReplaceAll(k.Exclude, string(c), "")
	}
	for _, s := range k.Contains {
		k.Exclude = strings.ReplaceAll(k.Exclude, s, "")
	}
}

func (k *Knowledge) Normalize() {
	k.Exclude = strings.ToUpper(k.Exclude)
	k.Exact = strings.ToUpper(k.Exact)
	for i, w := range k.Contains {
		k.Contains[i] = strings.ToUpper(w)
	}
}

func Suggest(k *Knowledge) ([]string, error) {
	results := make([]string, 0, 50)

	skipMatch := k.Exact == "....."
	matcher, err := regexp.Compile(k.Exact)
	if err != nil {
		return nil, fmt.Errorf("invalid exact match specification: %w", err)
	}

	containsCount := make(map[string]int)
	for _, l := range k.Contains {
		if len(l) != 1 {
			return nil, fmt.Errorf("invalid contains letter: %q", l)
		}
		if _, ok := containsCount[l]; !ok {
			containsCount[l] = 0
		}
		containsCount[l] = containsCount[l] + 1
	}

	for _, word := range dict.words {
		if !skipMatch {
			if m := matcher.FindString(word); m == "" {
				continue
			}
		}

		if strings.ContainsAny(word, k.Exclude) {
			continue
		}

		// must contain everything in the contains list
		addWord := true
		for mustContain, atLeast := range containsCount {
			if strings.Count(word, mustContain) < atLeast {
				addWord = false
				break
			}
		}

		// Filter out things we know to not be true
		if addWord {
			for _, exPattern := range k.NotInPosition {
				matcher, err := regexp.Compile(exPattern)
				if err != nil {
					continue
				}
				if m := matcher.FindString(word); m != "" {
					addWord = false
					break
				}
			}
		}

		if _, ok := k.PreviousWords[word]; ok {
			addWord = false
		}

		if addWord {
			results = append(results, word)
		}
	}

	return results, nil
}
