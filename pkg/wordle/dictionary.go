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
	"embed"
	"fmt"
	"strings"
)

//go:embed words5.txt
var f embed.FS

type Dictionary struct {
	words []string
}

func New() (*Dictionary, error) {
	data, err := f.ReadFile("words5.txt")
	if err != nil {
		return nil, fmt.Errorf("unable to load words: %w", err)
	}

	allWords := strings.Split(string(data), "\n")
	dict := make([]string, 0, len(allWords))
	for _, word := range allWords {
		word = strings.TrimSpace(word)
		// sanity check.
		if len(word) == 5 {
			dict = append(dict, strings.ToUpper(word))
		}
	}
	return &Dictionary{
		words: dict,
	}, nil
}
