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

import {
  ALPHABET,
  ContradictionError,
  CORRECT,
  PRESENT,
  WORD_LENGTH,
  assertPattern,
  assertWord,
} from './errors';

const UNKNOWN = -1;
const A = 65;

function letterName(index: number): string {
  return String.fromCharCode(A + index);
}

/**
 * Everything deduced about the hidden word so far.
 *
 * This mirrors `Knowledge` in pkg/wordle. State accumulates across turns: green
 * positions, per-letter count bounds, ruled-out positions and played words all
 * persist until {@link reset}.
 */
export class Knowledge {
  /** Letter index known to sit at each position, or -1. */
  private readonly greens = new Int8Array(WORD_LENGTH).fill(UNKNOWN);
  /** Bit `l` set means letter `l` cannot sit at this position. */
  private readonly banned = new Uint32Array(WORD_LENGTH);
  /** Inclusive bounds on how many times each letter appears. */
  private readonly minCount = new Uint8Array(ALPHABET);
  private readonly maxCount = new Uint8Array(ALPHABET).fill(WORD_LENGTH);
  private readonly played = new Set<string>();

  /**
   * The word whose feedback came back all green, or null.
   *
   * Worth recording separately because the winning word disappears from the
   * suggestions: it is a played word, so it is filtered out like any other, and
   * a solved puzzle looks exactly like an unsatisfiable one -- zero candidates.
   * Without this the UI cannot tell "you won" from "you mistyped a colour".
   */
  private solvedWith: string | null = null;

  /** Number of turns recorded. */
  get turns(): number {
    return this.played.size;
  }

  /** The winning word once a turn has come back all green. */
  get solved(): string | null {
    return this.solvedWith;
  }

  /** Forgets everything. */
  reset(): void {
    this.greens.fill(UNKNOWN);
    this.banned.fill(0);
    this.minCount.fill(0);
    this.maxCount.fill(WORD_LENGTH);
    this.played.clear();
    this.solvedWith = null;
  }

  /** Returns an independent copy. */
  clone(): Knowledge {
    const copy = new Knowledge();
    copy.greens.set(this.greens);
    copy.banned.set(this.banned);
    copy.minCount.set(this.minCount);
    copy.maxCount.set(this.maxCount);
    for (const word of this.played) copy.played.add(word);
    copy.solvedWith = this.solvedWith;
    return copy;
  }

  /** Reports whether `word` has already been played. */
  hasPlayed(word: string): boolean {
    return this.played.has(word.trim().toUpperCase());
  }

  /**
   * Folds one turn of feedback into this knowledge.
   *
   * Throws {@link InvalidWordError} or {@link InvalidPatternError} for
   * malformed input, and {@link ContradictionError} when the feedback cannot be
   * reconciled with earlier turns. In every failure case the knowledge is left
   * exactly as it was, so the caller can correct the entry and retry.
   */
  apply(guessInput: string, patternInput: string): void {
    const guess = assertWord(guessInput, 'guess');
    const pattern = assertPattern(patternInput);

    // Stage everything on copies; commit only once the whole turn checks out.
    const greens = Int8Array.from(this.greens);
    const banned = Uint32Array.from(this.banned);
    const minCount = Uint8Array.from(this.minCount);
    const maxCount = Uint8Array.from(this.maxCount);

    // Per-letter tallies for this turn alone. `seen` counts green and yellow
    // tiles, `absent` counts gray ones. The bounds they imply are only valid
    // within one self-consistent piece of feedback.
    const seen = new Uint8Array(ALPHABET);
    const absent = new Uint8Array(ALPHABET);

    for (let i = 0; i < WORD_LENGTH; i++) {
      const letter = guess.charCodeAt(i) - A;
      const mark = pattern[i];
      const position = i + 1;

      if (mark === CORRECT) {
        const prev = greens[i]!;
        if (prev !== UNKNOWN && prev !== letter) {
          throw new ContradictionError(
            `position ${position} is already known to be ${letterName(prev)}, it cannot also be ${letterName(letter)}`,
          );
        }
        if ((banned[i]! & (1 << letter)) !== 0) {
          throw new ContradictionError(
            `${letterName(letter)} is already known not to be at position ${position}`,
          );
        }
        greens[i] = letter;
        seen[letter] = seen[letter]! + 1;
        continue;
      }

      if (greens[i] === letter) {
        throw new ContradictionError(
          `${letterName(letter)} is already known to be at position ${position}, so it cannot be marked ` +
            (mark === PRESENT ? 'misplaced' : 'absent') +
            ' there',
        );
      }
      banned[i] = banned[i]! | (1 << letter);
      if (mark === PRESENT) {
        seen[letter] = seen[letter]! + 1;
      } else {
        absent[letter] = absent[letter]! + 1;
      }
    }

    for (let l = 0; l < ALPHABET; l++) {
      if (seen[l]! > minCount[l]!) {
        minCount[l] = seen[l]!;
      }
      // A gray tile proves the word holds no more of that letter than the green
      // and yellow tiles in the same guess already account for. This is what
      // turns SASSY against MOSSY into "exactly two S" rather than "at least
      // one S".
      if (absent[l]! > 0 && seen[l]! < maxCount[l]!) {
        maxCount[l] = seen[l]!;
      }
      if (minCount[l]! > maxCount[l]!) {
        throw new ContradictionError(
          `${letterName(l)} would have to appear at least ${minCount[l]} and at most ${maxCount[l]} times`,
        );
      }
    }

    this.greens.set(greens);
    this.banned.set(banned);
    this.minCount.set(minCount);
    this.maxCount.set(maxCount);
    this.played.add(guess);
    if (pattern === CORRECT.repeat(WORD_LENGTH)) {
      this.solvedWith = guess;
    }
  }

  /**
   * Reports whether `word` is consistent with everything known.
   *
   * `word` is expected to be five uppercase letters; anything else never
   * matches.
   */
  match(word: string): boolean {
    if (word.length !== WORD_LENGTH) return false;

    const counts = new Uint8Array(ALPHABET);
    for (let i = 0; i < WORD_LENGTH; i++) {
      const letter = word.charCodeAt(i) - A;
      if (letter < 0 || letter >= ALPHABET) return false;

      const green = this.greens[i]!;
      if (green !== UNKNOWN && green !== letter) return false;
      if ((this.banned[i]! & (1 << letter)) !== 0) return false;
      counts[letter] = counts[letter]! + 1;
    }

    for (let l = 0; l < ALPHABET; l++) {
      const n = counts[l]!;
      if (n < this.minCount[l]! || n > this.maxCount[l]!) return false;
    }

    return !this.played.has(word);
  }

  /** Renders the state for debugging, in the same shape as the Go `String()`. */
  describe(): string {
    let greens = '';
    for (let i = 0; i < WORD_LENGTH; i++) {
      greens += this.greens[i] === UNKNOWN ? '.' : letterName(this.greens[i]!);
    }

    const counts: string[] = [];
    for (let l = 0; l < ALPHABET; l++) {
      if (this.minCount[l] === 0 && this.maxCount[l] === WORD_LENGTH) continue;
      counts.push(`${letterName(l)}:${this.minCount[l]}-${this.maxCount[l]}`);
    }

    const banned: string[] = [];
    for (let i = 0; i < WORD_LENGTH; i++) {
      if (this.banned[i] === 0) continue;
      let letters = '';
      for (let l = 0; l < ALPHABET; l++) {
        if ((this.banned[i]! & (1 << l)) !== 0) letters += letterName(l);
      }
      banned.push(`${i + 1}:${letters}`);
    }

    const solved = this.solvedWith === null ? '' : ` solved=${this.solvedWith}`;
    return `greens=${greens} counts=[${counts.join(' ')}] banned=[${banned.join(' ')}] guessed=${this.played.size}${solved}`;
  }
}
