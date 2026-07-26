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

import { ABSENT, ALPHABET, CORRECT, PRESENT, WORD_LENGTH, assertWord } from './errors';

/** Mark values, ordered so they double as base-3 digits. */
export const MARK_ABSENT = 0;
export const MARK_PRESENT = 1;
export const MARK_CORRECT = 2;

/** Number of distinct feedback patterns: 3 marks over 5 positions. */
export const PATTERN_COUNT = 243;

const MARK_CHARS = [ABSENT, PRESENT, CORRECT] as const;

/** Encodes a word as letter indices 0-25. */
export function encodeWord(word: string): Uint8Array {
  const out = new Uint8Array(WORD_LENGTH);
  for (let i = 0; i < WORD_LENGTH; i++) {
    out[i] = word.charCodeAt(i) - 65;
  }
  return out;
}

// Scratch buffers, reused across calls. Ranking evaluates patternCode millions
// of times per pass, and allocating two small arrays each time dominates the
// actual work. Safe because JavaScript is single-threaded within a worker and
// patternCode neither yields nor recurses.
const marks = new Uint8Array(WORD_LENGTH);
const unclaimed = new Uint8Array(ALPHABET);

/**
 * Returns the base-3 code (0-242) of the feedback for `guess` against `answer`,
 * where both are letter indices and position 0 is the least significant digit.
 *
 * Greens are resolved before yellows, and each yellow consumes one of the
 * answer's remaining letters. That ordering is the whole subtlety of Wordle
 * feedback: guessing SPEED against ABIDE marks the first E misplaced and the
 * second absent, because ABIDE has only one E to give.
 */
export function patternCode(
  guess: Uint8Array,
  guessOffset: number,
  answer: Uint8Array,
  answerOffset: number,
): number {
  for (let i = 0; i < WORD_LENGTH; i++) {
    const g = guess[guessOffset + i]!;
    const a = answer[answerOffset + i]!;
    if (g === a) {
      marks[i] = MARK_CORRECT;
    } else {
      marks[i] = MARK_ABSENT;
      unclaimed[a] = unclaimed[a]! + 1;
    }
  }

  for (let i = 0; i < WORD_LENGTH; i++) {
    if (marks[i] === MARK_CORRECT) continue;
    const g = guess[guessOffset + i]!;
    if (unclaimed[g]! > 0) {
      marks[i] = MARK_PRESENT;
      unclaimed[g] = unclaimed[g]! - 1;
    }
  }

  // Reset only the entries this call touched, which is cheaper than clearing
  // all 26 and is why `unclaimed` can be shared.
  for (let i = 0; i < WORD_LENGTH; i++) {
    unclaimed[answer[answerOffset + i]!] = 0;
  }

  return marks[0]! + marks[1]! * 3 + marks[2]! * 9 + marks[3]! * 27 + marks[4]! * 81;
}

/** Renders a base-3 pattern code as a `.YG` string. */
export function codeToPattern(code: number): string {
  let out = '';
  let rest = code;
  for (let i = 0; i < WORD_LENGTH; i++) {
    out += MARK_CHARS[rest % 3]!;
    rest = Math.floor(rest / 3);
  }
  return out;
}

/** Parses a `.YG` string into its base-3 pattern code. */
export function patternToCode(pattern: string): number {
  let code = 0;
  for (let i = WORD_LENGTH - 1; i >= 0; i--) {
    const ch = pattern[i];
    code = code * 3 + (ch === CORRECT ? MARK_CORRECT : ch === PRESENT ? MARK_PRESENT : MARK_ABSENT);
  }
  return code;
}

/**
 * Returns the feedback Wordle would show for `guess` against `answer`, as a
 * `.YG` string.
 *
 * Throws {@link InvalidWordError} if either argument is not five letters.
 */
export function feedback(guess: string, answer: string): string {
  const g = assertWord(guess, 'guess');
  const a = assertWord(answer, 'answer');
  return codeToPattern(patternCode(encodeWord(g), 0, encodeWord(a), 0));
}
