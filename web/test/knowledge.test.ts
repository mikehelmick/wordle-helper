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

import { WORDS } from '../src/data/words';
import {
  ContradictionError,
  InvalidPatternError,
  InvalidWordError,
} from '../src/solver/errors';
import { feedback } from '../src/solver/feedback';
import { suggest } from '../src/solver/filter';
import { Knowledge } from '../src/solver/knowledge';

function knowing(...turns: Array<[string, string]>): Knowledge {
  const k = new Knowledge();
  for (const [guess, pattern] of turns) k.apply(guess, pattern);
  return k;
}

describe('Knowledge', () => {
  it('matches everything before any turn', () => {
    const k = new Knowledge();
    for (const word of ['CRANE', 'TOAST', 'SASSY', 'ZYMIC']) {
      expect(k.match(word)).toBe(true);
    }
  });

  it('keeps green positions across later turns', () => {
    const k = knowing(['CRANE', '..G..'], ['MILKY', '.....']);
    expect(k.match('TOAST')).toBe(true);
    expect(k.match('ABOUT')).toBe(false); // A is pinned to position 3
    expect(k.match('TOADY')).toBe(false); // Y was ruled out in turn two
  });

  it('caps a letter count when a duplicate comes back gray', () => {
    const k = knowing(['ERASE', '..Y.G']);
    expect(k.match('ABIDE')).toBe(true);
    expect(k.match('AGLEE')).toBe(false); // exactly one E, and AGLEE has two
  });

  it('rules out the position of a misplaced letter', () => {
    const k = knowing(['CRANE', '..Y..']);
    expect(k.match('TOAST')).toBe(false); // A cannot stay in the middle
    expect(k.match('ADOPT')).toBe(true);
    expect(k.match('SHOUT')).toBe(false); // no A at all
  });

  it('never suggests a word already played', () => {
    const k = knowing(['ABIDE', 'GGGGG']);
    expect(k.hasPlayed('ABIDE')).toBe(true);
    expect(k.match('ABIDE')).toBe(false);
  });

  it('normalizes case and surrounding whitespace', () => {
    expect(knowing([' crane ', ' ..g.. ']).describe()).toBe(knowing(['CRANE', '..G..']).describe());
  });

  describe('rejects contradictory feedback', () => {
    const cases: Array<[string, Array<[string, string]>]> = [
      ['two different letters green in one position', [['CRANE', 'G....'], ['TOAST', 'G....']]],
      ['a letter green where it was ruled out', [['CRANE', '.Y...'], ['TRACE', '.G...']]],
      ['a known green reported misplaced', [['CRANE', 'G....'], ['CHOMP', 'Y....']]],
      ['a known green reported absent', [['CRANE', 'G....'], ['CHOMP', '.....']]],
      ['a letter required and forbidden at once', [['CRANE', '..Y..'], ['ABIDE', '.....']]],
    ];

    for (const [name, turns] of cases) {
      it(name, () => {
        const k = knowing(...turns.slice(0, -1));
        const before = k.describe();
        const [guess, pattern] = turns[turns.length - 1]!;

        expect(() => k.apply(guess, pattern)).toThrow(ContradictionError);
        // A rejected turn must be a no-op, or the user cannot simply re-enter it.
        expect(k.describe()).toBe(before);
      });
    }
  });

  describe('rejects malformed input', () => {
    const cases: Array<[string, string, string, unknown]> = [
      ['short guess', 'CRAN', '.....', InvalidWordError],
      ['long guess', 'CRANES', '.....', InvalidWordError],
      ['digits in guess', 'CR4NE', '.....', InvalidWordError],
      ['accented letter', 'CRÁNE', '.....', InvalidWordError],
      ['short pattern', 'CRANE', '....', InvalidPatternError],
      ['unknown pattern character', 'CRANE', '..X..', InvalidPatternError],
      // These were handed to a regular expression compiler by the original Go
      // implementation. Both are exactly five characters, so a length check
      // alone let them through.
      ['regex character class as guess', '[A-Z]', '.....', InvalidWordError],
      ['regex quantifier as pattern', 'CRANE', '.*.*.', InvalidPatternError],
      ['regex anchors as pattern', 'CRANE', '^...$', InvalidPatternError],
    ];

    for (const [name, guess, pattern, expected] of cases) {
      it(name, () => {
        const k = new Knowledge();
        const before = k.describe();
        expect(() => k.apply(guess, pattern)).toThrow(expected as never);
        expect(k.describe()).toBe(before);
      });
    }
  });

  it('clones independently', () => {
    const original = knowing(['CRANE', '..G..']);
    const copy = original.clone();
    copy.apply('MILKY', '.....');

    expect(original.match('TOADY')).toBe(true);
    expect(copy.match('TOADY')).toBe(false);
  });

  it('reset returns to a blank slate', () => {
    const k = knowing(['CRANE', '..G..']);
    k.reset();
    expect(k.describe()).toBe(new Knowledge().describe());
    expect(k.match('ZYMIC')).toBe(true);
  });
});

describe('solving real games', () => {
  // Duplicate letters are where the filtering rules are hardest to get right.
  const answers = ['TOAST', 'SASSY', 'ABBEY', 'LEVEL', 'SPEED', 'MUMMY', 'EERIE', 'CRANE'];

  for (const answer of answers) {
    it(`never filters out ${answer}`, () => {
      const k = new Knowledge();
      let guess = 'CRANE';

      for (let round = 0; round < 8 && guess !== answer; round++) {
        const pattern = feedback(guess, answer);
        k.apply(guess, pattern);

        const possible = suggest(k);
        expect(possible, `after guessing ${guess} (${pattern}); ${k.describe()}`).toContain(answer);
        guess = possible[0]!;
      }
    });
  }

  it('agrees with the CLI transcript in the README', () => {
    const k = knowing(['CRANE', '..G..']);
    expect(suggest(k)).toHaveLength(323);
    k.apply('ABAFT', '..G.G');
    expect(suggest(k)).toHaveLength(15);
    k.apply('GHAST', '..GGG');
    expect(suggest(k)).toEqual(['LOAST', 'PLAST', 'TOAST']);
  });

  it('ships the same word list as the Go package', () => {
    expect(WORDS).toHaveLength(12972);
    expect(WORDS[0]).toBe('AAHED');
    expect(WORDS[WORDS.length - 1]).toBe('ZYMIC');
    expect([...WORDS].sort()).toEqual([...WORDS]);
  });
});
