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

import { PACKED_LETTERS, WORDS, WORD_LENGTH } from '../data/words';
import { PATTERN_COUNT, patternCode } from './feedback';

/** A guess scored by how much it is expected to narrow the candidate set. */
export interface RankedGuess {
  /** The word to play. */
  readonly word: string;
  /** Expected information gain, in bits. */
  readonly bits: number;
  /** Whether this word is itself still a possible answer. */
  readonly candidate: boolean;
}

export interface RankOptions {
  /** How many guesses to return. Defaults to 20. */
  readonly limit?: number;
  /**
   * Restrict scoring to the candidates themselves rather than the full word
   * list. Much faster, and the right trade when only a handful remain.
   */
  readonly candidatesOnly?: boolean;
}

/**
 * The candidate-set size past which a full sweep stops being interactive.
 *
 * Cost is `candidates x 12,972` pattern evaluations. A few hundred candidates
 * scores in well under a second; the opening set of 12,972 would be 168 million
 * evaluations, which is why the first guess is precomputed instead.
 */
export const LIVE_RANK_LIMIT = 3000;

/**
 * Ranks guesses by expected information gain against `candidates`.
 *
 * For a guess `g`, partition the candidates by the feedback `g` would produce.
 * A guess that scatters them across many small groups tells you more than one
 * that leaves a single large group, and the entropy of that partition,
 * `-sum(p * log2 p)`, is exactly "expected bits learned".
 *
 * Every word in the list is scored by default, not just the candidates: with
 * five plausible answers left, a word that cannot be the answer but separates
 * them cleanly is often the better play.
 */
export function rankGuesses(candidates: Int32Array, options: RankOptions = {}): RankedGuess[] {
  const limit = options.limit ?? 20;
  const n = candidates.length;
  if (n === 0) return [];

  const candidateSet = new Set<number>();
  for (let i = 0; i < n; i++) candidateSet.add(candidates[i]!);

  // One shared bucket table, cleared by touching only the patterns each guess
  // actually produced -- at most `n` of the 243, and usually far fewer.
  const buckets = new Int32Array(PATTERN_COUNT);
  const touched = new Int32Array(PATTERN_COUNT);
  const invN = 1 / n;

  const pool = options.candidatesOnly ? candidates : null;
  const poolSize = pool ? pool.length : WORDS.length;
  const scored: RankedGuess[] = new Array(poolSize);

  for (let p = 0; p < poolSize; p++) {
    const g = pool ? pool[p]! : p;
    const guessOffset = g * WORD_LENGTH;

    let touchedCount = 0;
    for (let c = 0; c < n; c++) {
      const code = patternCode(PACKED_LETTERS, guessOffset, PACKED_LETTERS, candidates[c]! * WORD_LENGTH);
      if (buckets[code] === 0) touched[touchedCount++] = code;
      buckets[code] = buckets[code]! + 1;
    }

    let bits = 0;
    for (let t = 0; t < touchedCount; t++) {
      const code = touched[t]!;
      const probability = buckets[code]! * invN;
      bits -= probability * Math.log2(probability);
      buckets[code] = 0;
    }

    scored[p] = { word: WORDS[g]!, bits, candidate: candidateSet.has(g) };
  }

  scored.sort(compareRanked);
  return scored.slice(0, limit);
}

/**
 * Orders by information first, then prefers a word that could itself be the
 * answer: given two equally informative guesses, one of them can win outright.
 * Alphabetical order breaks the remaining ties so results are stable.
 */
function compareRanked(a: RankedGuess, b: RankedGuess): number {
  if (b.bits !== a.bits) return b.bits - a.bits;
  if (a.candidate !== b.candidate) return a.candidate ? -1 : 1;
  return a.word < b.word ? -1 : a.word > b.word ? 1 : 0;
}
