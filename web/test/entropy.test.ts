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

import { describe, expect, it } from 'vitest';

import { PACKED_LETTERS, WORDS, WORD_LENGTH } from '../src/data/words';
import { rankGuesses } from '../src/solver/entropy';
import {
  PATTERN_COUNT,
  codeToPattern,
  encodeWord,
  feedback,
  patternCode,
  patternToCode,
} from '../src/solver/feedback';

const index = new Map(WORDS.map((word, i) => [word, i]));

function indicesOf(...words: string[]): Int32Array {
  return Int32Array.from(
    words.map((word) => {
      const i = index.get(word);
      if (i === undefined) throw new Error(`${word} is not in the word list`);
      return i;
    }),
  );
}

describe('pattern codes', () => {
  it('round-trips every possible pattern', () => {
    for (let code = 0; code < PATTERN_COUNT; code++) {
      const pattern = codeToPattern(code);
      expect(pattern).toHaveLength(WORD_LENGTH);
      expect(patternToCode(pattern)).toBe(code);
    }
  });

  it('encodes the extremes as 0 and 242', () => {
    expect(patternToCode('.....')).toBe(0);
    expect(patternToCode('GGGGG')).toBe(PATTERN_COUNT - 1);
  });

  it('agrees with the string-level feedback function', () => {
    // The packed array is the hot path; the string API is what tests and the
    // UI use. They must not diverge.
    for (const [guess, answer] of [
      ['CRANE', 'TOAST'],
      ['SPEED', 'ABIDE'],
      ['SASSY', 'MOSSY'],
      ['ALLEY', 'LEVEL'],
      ['MUMMY', 'MOTTO'],
    ] as const) {
      const viaPacked = patternCode(
        PACKED_LETTERS,
        index.get(guess)! * WORD_LENGTH,
        PACKED_LETTERS,
        index.get(answer)! * WORD_LENGTH,
      );
      expect(codeToPattern(viaPacked)).toBe(feedback(guess, answer));
    }
  });

  it('does not leak state between calls through its scratch buffers', () => {
    const first = feedback('SASSY', 'MOSSY');
    for (const [guess, answer] of [
      ['MUMMY', 'MOTTO'],
      ['EERIE', 'SPEED'],
      ['LEVEL', 'ALLEY'],
    ] as const) {
      feedback(guess, answer);
    }
    expect(feedback('SASSY', 'MOSSY')).toBe(first);
  });

  it('encodes words as zero-based letter indices', () => {
    expect(Array.from(encodeWord('ABCDE'))).toEqual([0, 1, 2, 3, 4]);
  });
});

describe('rankGuesses', () => {
  it('returns nothing when no candidate remains', () => {
    expect(rankGuesses(Int32Array.from([]))).toEqual([]);
  });

  it('scores a single candidate at zero bits and ranks it first', () => {
    const ranked = rankGuesses(indicesOf('TOAST'), { limit: 3 });
    // One candidate means nothing left to learn, so every guess scores 0 and
    // the tie-break must surface the word that actually wins the game.
    expect(ranked[0]!.word).toBe('TOAST');
    expect(ranked[0]!.candidate).toBe(true);
    expect(ranked[0]!.bits).toBeCloseTo(0, 12);
  });

  it('extracts exactly one bit from two candidates', () => {
    const ranked = rankGuesses(indicesOf('ABIDE', 'ABBEY'), { limit: 5 });
    // Splitting two equally likely answers is one bit, and no guess can do
    // better than telling them apart.
    expect(ranked[0]!.bits).toBeCloseTo(1, 12);
  });

  it('never exceeds the information available', () => {
    const candidates = indicesOf('BOSSY', 'FUSSY', 'GUSSY', 'HISSY', 'MESSY', 'MOSSY', 'MUSSY', 'PUSSY');
    const ceiling = Math.log2(candidates.length);
    for (const guess of rankGuesses(candidates, { limit: 40 })) {
      expect(guess.bits).toBeLessThanOrEqual(ceiling + 1e-9);
      expect(guess.bits).toBeGreaterThanOrEqual(0);
    }
  });

  it('prefers a non-candidate probe when it separates better', () => {
    // These eight differ only in their first letter, so guessing one of them
    // can only ever confirm or reject that single word: one outcome of
    // probability 1/8 and one of 7/8, worth 0.5436 bits. A probe carrying
    // several of the eight distinguishing letters does far better.
    //
    // The ceiling here is 2 bits, not log2(8) = 3: telling all eight apart at
    // once would need a guess holding five of B, D, F, G, H, M, P and T, and no
    // English word is built only from those. The best available, such as BUMPH
    // or DEPTH, carry four, which isolates four candidates and leaves the other
    // four pooled.
    const candidates = indicesOf('BILLS', 'DILLS', 'FILLS', 'GILLS', 'HILLS', 'MILLS', 'PILLS', 'TILLS');
    const best = rankGuesses(candidates, { limit: 1 })[0]!;
    const bestCandidateOnly = rankGuesses(candidates, { limit: 1, candidatesOnly: true })[0]!;

    expect(best.candidate).toBe(false);
    expect(best.bits).toBeCloseTo(2, 12);

    expect(bestCandidateOnly.candidate).toBe(true);
    expect(bestCandidateOnly.bits).toBeCloseTo((1 / 8) * 3 + (7 / 8) * Math.log2(8 / 7), 12);
    expect(best.bits).toBeGreaterThan(bestCandidateOnly.bits);
  });

  it('honours the requested limit', () => {
    expect(rankGuesses(indicesOf('ABIDE', 'ABBEY', 'TOAST'), { limit: 7 })).toHaveLength(7);
  });

  it('returns results sorted by descending information', () => {
    const ranked = rankGuesses(indicesOf('BOSSY', 'FUSSY', 'MESSY', 'MOSSY', 'MUSSY'), { limit: 25 });
    for (let i = 1; i < ranked.length; i++) {
      expect(ranked[i - 1]!.bits).toBeGreaterThanOrEqual(ranked[i]!.bits);
    }
  });

  it('is deterministic', () => {
    const candidates = indicesOf('BOSSY', 'FUSSY', 'MESSY', 'MOSSY');
    expect(rankGuesses(candidates, { limit: 10 })).toEqual(rankGuesses(candidates, { limit: 10 }));
  });
});
