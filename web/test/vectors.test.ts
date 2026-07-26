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

// The solver exists twice: once in Go, once here. Two implementations of one
// algorithm drift, and the drift stays invisible until somebody gets a wrong
// answer. This suite asserts against the exact file the Go tests generate, so
// any divergence fails the build on both sides.
//
// Regenerate after a deliberate behaviour change:
//   go test ./pkg/wordle -run TestVectors -update

import { readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

import { WORDS } from '../src/data/words';
import { feedback } from '../src/solver/feedback';
import { suggest } from '../src/solver/filter';
import { Knowledge } from '../src/solver/knowledge';

interface FeedbackCase {
  name: string;
  guess: string;
  answer: string;
  pattern: string;
}

interface Turn {
  guess: string;
  pattern: string;
}

interface SolveCase {
  name: string;
  dictionary?: string[];
  turns: Turn[];
  expected: string[];
}

interface VectorFile {
  note: string;
  wordLen: number;
  feedback: FeedbackCase[];
  solve: SolveCase[];
}

const HERE = dirname(fileURLToPath(import.meta.url));
const VECTORS_PATH = resolve(HERE, '../../pkg/wordle/testdata/vectors.json');

const vectors = JSON.parse(readFileSync(VECTORS_PATH, 'utf8')) as VectorFile;

describe('shared Go/TypeScript vectors', () => {
  it('loads a non-empty vector file', () => {
    expect(vectors.wordLen).toBe(5);
    expect(vectors.feedback.length).toBeGreaterThan(0);
    expect(vectors.solve.length).toBeGreaterThan(0);
  });

  describe('feedback', () => {
    for (const testCase of vectors.feedback) {
      it(`${testCase.name}: ${testCase.guess} vs ${testCase.answer}`, () => {
        expect(feedback(testCase.guess, testCase.answer)).toBe(testCase.pattern);
      });
    }
  });

  describe('solve', () => {
    for (const testCase of vectors.solve) {
      it(testCase.name, () => {
        // Go's NewDictionary sorts and deduplicates; match that so the output
        // order is comparable. Cases with no dictionary run against the full
        // embedded list, which both sides generate from the same source file.
        const words = testCase.dictionary
          ? [...new Set(testCase.dictionary.map((w) => w.trim().toUpperCase()))].sort()
          : WORDS;

        const knowledge = new Knowledge();
        for (const turn of testCase.turns) {
          knowledge.apply(turn.guess, turn.pattern);
        }

        expect(suggest(knowledge, words)).toEqual(testCase.expected);
      });
    }
  });
});
