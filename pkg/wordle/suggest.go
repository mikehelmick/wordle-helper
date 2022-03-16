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

type Knowledge struct {
	Exclude  string
	Contains []string
	Exact    string
}

func NewKnowledge() *Knowledge {
	return &Knowledge{
		Exclude:  "",
		Contains: make([]string, 0),
		Exact:    "",
	}
}

func (k *Knowledge) CleanExclude() {
	for _, c := range k.Exact {
		k.Exclude = strings.ReplaceAll(k.Exclude, string(c), "")
	}
	for _, s := range k.Contains {
		k.Exclude = strings.ReplaceAll(k.Exclude, s, "")
	}
}

var (
	dict *Dictionary
)

func init() {
	var err error
	dict, err = New()
	if err != nil {
		panic(err)
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

		if addWord {
			results = append(results, word)
		}
	}

	return results, nil
}
