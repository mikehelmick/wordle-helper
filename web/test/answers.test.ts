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

import { readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

import { ANSWERS, ANSWER_COUNT, IS_ANSWER, WORDS } from '../src/data/words';
import { onlyAnswers, suggestIndices, toWords } from '../src/solver/filter';
import { Knowledge } from '../src/solver/knowledge';

const HERE = dirname(fileURLToPath(import.meta.url));
const GO_ANSWERS = resolve(HERE, '../../pkg/wordle/answers.go');

describe('the answer list', () => {
  it('matches the list the Go solver compiles in', () => {
    // Same anti-drift rule as the word list: one source, checked rather than
    // trusted. A silent difference here would make the two implementations
    // rank differently while both looked correct.
    const fromGo = [...readFileSync(GO_ANSWERS, 'utf8').matchAll(/"([A-Z]{5})"/g)].map(
      (match) => match[1] as string,
    );

    expect(fromGo).toHaveLength(ANSWER_COUNT);
    expect([...ANSWERS].sort()).toEqual([...fromGo].sort());
  });

  it('is a strict subset of the legal guesses', () => {
    expect(ANSWER_COUNT).toBe(2315);
    expect(ANSWER_COUNT).toBeLessThan(WORDS.length);
    for (const answer of ANSWERS) {
      expect(WORDS, `${answer} is not a legal guess`).toContain(answer);
    }
  });

  it('keeps IS_ANSWER aligned with the set', () => {
    expect(IS_ANSWER).toHaveLength(WORDS.length);
    let flagged = 0;
    for (let i = 0; i < WORDS.length; i++) {
      const expected = ANSWERS.has(WORDS[i]!) ? 1 : 0;
      if (IS_ANSWER[i] !== expected) {
        throw new Error(`IS_ANSWER[${i}] disagrees for ${WORDS[i]}`);
      }
      flagged += IS_ANSWER[i]!;
    }
    expect(flagged).toBe(ANSWER_COUNT);
  });

  it('separates real answers from merely legal guesses', () => {
    expect(ANSWERS.has('TOAST')).toBe(true);
    expect(ANSWERS.has('CRANE')).toBe(true);
    // Legal to type, but they have never been and will never be the answer.
    // Telling these apart is the whole reason the list ships.
    for (const word of ['LOAST', 'PLAST', 'QAPIK', 'ZYMIC', 'PIKAU']) {
      expect(ANSWERS.has(word), `${word} should not be a possible answer`).toBe(false);
    }
  });
});

describe('filtering to plausible answers', () => {
  it('narrows the worked example to a single word', () => {
    const knowledge = new Knowledge();
    knowledge.apply('CRANE', '..G..');
    knowledge.apply('ABAFT', '..G.G');
    knowledge.apply('GHAST', '..GGG');

    const all = suggestIndices(knowledge);
    // Nothing is discarded: all three still survive the filter.
    expect(toWords(all)).toEqual(['LOAST', 'PLAST', 'TOAST']);
    // But only one of them has ever been an answer.
    expect(toWords(onlyAnswers(all))).toEqual(['TOAST']);
  });

  it('returns nothing when no survivor has ever been an answer', () => {
    const knowledge = new Knowledge();
    knowledge.apply('SASSY', '..GGG');

    const all = suggestIndices(knowledge);
    const likely = onlyAnswers(all);
    expect(all.length).toBeGreaterThan(likely.length);
    // The caller has to cope with this: it means the answer is a word outside
    // the known list, and ranking must fall back to the full candidate set.
    expect(toWords(likely).every((w) => ANSWERS.has(w))).toBe(true);
  });

  it('preserves dictionary order', () => {
    const knowledge = new Knowledge();
    knowledge.apply('CRANE', '..G..');
    const likely = toWords(onlyAnswers(suggestIndices(knowledge)));
    expect(likely).toEqual([...likely].sort());
  });
});
