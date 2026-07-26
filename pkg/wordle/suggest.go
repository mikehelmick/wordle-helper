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

import "errors"

// Suggest returns every word in d consistent with k, in dictionary order.
//
// Matching is done by direct index comparison. A previous implementation
// compiled the caller's constraints into regular expressions -- once per
// constraint, per word, per call -- which made user keystrokes an injection
// surface and did far more work than the task requires.
func Suggest(d *Dictionary, k *Knowledge) ([]string, error) {
	if d == nil {
		return nil, errors.New("wordle: nil dictionary")
	}
	if k == nil {
		return nil, errors.New("wordle: nil knowledge")
	}

	results := make([]string, 0, 64)
	for _, word := range d.Words {
		if k.Match(word) {
			results = append(results, word)
		}
	}
	return results, nil
}
