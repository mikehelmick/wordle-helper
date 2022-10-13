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

	"github.com/mikehelmick/wordle-helper/pkg/wordle"
)

func main() {
	k := &wordle.Knowledge{
		Exclude: "CHIRMKESBL",
		Contains: []string{
			"O",
			"T",
		},
		Exact: "...A.",
	}
	fmt.Printf("Suggest input: %+v\n", *k)

	possible, err := wordle.Suggest(k)
	if err != nil {
		fmt.Printf("bad input: %v\n", err)
	}

	fmt.Printf("FOUND %v POSSIBLE MATCHES\n", len(possible))
	output := strings.Join(possible, " ")
	fmt.Printf("%s\n", output)
}
