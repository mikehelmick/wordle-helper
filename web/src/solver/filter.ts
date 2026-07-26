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

import { WORDS } from '../data/words';
import type { Knowledge } from './knowledge';

/**
 * Returns every word in `words` consistent with `knowledge`, in the order given
 * (the shipped list is sorted, so output is alphabetical).
 *
 * Port of `Suggest` in pkg/wordle. Matching is direct comparison: no regular
 * expressions are built from user input.
 */
export function suggest(knowledge: Knowledge, words: readonly string[] = WORDS): string[] {
  const out: string[] = [];
  for (const word of words) {
    if (knowledge.match(word)) out.push(word);
  }
  return out;
}

/**
 * Same filter, but returning indices into {@link WORDS}.
 *
 * Ranking scores candidates straight out of the packed letter array, so it
 * wants positions rather than strings.
 */
export function suggestIndices(knowledge: Knowledge): Int32Array {
  const out: number[] = [];
  for (let i = 0; i < WORDS.length; i++) {
    if (knowledge.match(WORDS[i]!)) out.push(i);
  }
  return Int32Array.from(out);
}
