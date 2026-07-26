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

/** Number of letters in every word. */
export const WORD_LENGTH = 5;

/** Number of distinct letters a word may be built from. */
export const ALPHABET = 26;

/** Feedback characters, matching the Go package and the CLI. */
export const ABSENT = '.';
export const PRESENT = 'Y';
export const CORRECT = 'G';

/** A guess that is not exactly five uppercase ASCII letters. */
export class InvalidWordError extends Error {
  override readonly name = 'InvalidWordError';
}

/** A feedback string that is not five characters drawn from `.`, `Y` and `G`. */
export class InvalidPatternError extends Error {
  override readonly name = 'InvalidPatternError';
}

/** Feedback that cannot be reconciled with what is already known. */
export class ContradictionError extends Error {
  override readonly name = 'ContradictionError';
}

const WORD_RE = /^[A-Z]{5}$/;
const PATTERN_RE = /^[.YG]{5}$/;

/** Trims and upper-cases caller input before it is validated. */
export function normalize(s: string): string {
  return s.trim().toUpperCase();
}

/**
 * Returns `s` normalized, or throws {@link InvalidWordError}.
 *
 * The regex is anchored and character-class based rather than a length check:
 * `'CRÁN'.length` is 4 in JavaScript but the Go side sees 5 bytes, and only an
 * explicit A-Z test rejects it identically in both.
 */
export function assertWord(s: string, what = 'word'): string {
  const word = normalize(s);
  if (!WORD_RE.test(word)) {
    throw new InvalidWordError(`invalid ${what}: ${JSON.stringify(s)} must be exactly ${WORD_LENGTH} letters A-Z`);
  }
  return word;
}

/** Returns `s` normalized, or throws {@link InvalidPatternError}. */
export function assertPattern(s: string): string {
  const pattern = normalize(s);
  if (!PATTERN_RE.test(pattern)) {
    throw new InvalidPatternError(
      `invalid pattern: ${JSON.stringify(s)} must be ${WORD_LENGTH} characters, each one of ` +
        `${ABSENT} (not in word), ${PRESENT} (wrong spot) or ${CORRECT} (correct)`,
    );
  }
  return pattern;
}
