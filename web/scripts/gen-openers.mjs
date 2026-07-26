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

// Precomputes the best opening guesses and writes src/data/openers.json.
//
// Before the first guess every word is still possible, so ranking openers means
// scoring all 12,972 guesses against all 12,972 candidates -- 168 million
// pattern evaluations. That is far too slow to do in the browser while someone
// waits, and the answer never changes, so it is computed once here and shipped
// as data. Every later turn has a small enough candidate set to rank live.
//
//   node scripts/gen-openers.mjs            # all cores, top 20
//   node scripts/gen-openers.mjs --top 50
//
// Roughly a second on ten cores, so it is cheap to re-run, but it stays out of
// the build: the output only changes when the word list does.

import { availableParallelism } from 'node:os';
import { readFileSync, writeFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { Worker, isMainThread, parentPort, workerData } from 'node:worker_threads';

const HERE = dirname(fileURLToPath(import.meta.url));
const SOURCE = resolve(HERE, '../../pkg/wordle/words5.txt');
const TARGET = resolve(HERE, '../src/data/openers.json');

const WORD_LENGTH = 5;
const ALPHABET = 26;
const PATTERN_COUNT = 243;

function loadWords() {
  const words = [];
  const seen = new Set();
  for (const line of readFileSync(SOURCE, 'utf8').split('\n')) {
    const word = line.trim().toUpperCase();
    if (word === '' || seen.has(word)) continue;
    if (!/^[A-Z]{5}$/.test(word)) throw new Error(`invalid entry ${JSON.stringify(word)}`);
    seen.add(word);
    words.push(word);
  }
  return words.sort();
}

function pack(words) {
  const out = new Uint8Array(words.length * WORD_LENGTH);
  for (let i = 0; i < words.length; i++) {
    for (let j = 0; j < WORD_LENGTH; j++) {
      out[i * WORD_LENGTH + j] = words[i].charCodeAt(j) - 65;
    }
  }
  return out;
}

/**
 * Scores guesses [from, to) against every word, returning {index, bits}.
 *
 * Mirrors src/solver/feedback.ts and src/solver/entropy.ts. The logic is
 * duplicated rather than imported because this script is plain ESM and the
 * solver is TypeScript; the vitest suite pins both to the same shared vectors,
 * so a divergence here would show up as a failing test rather than a silently
 * wrong opener.
 */
function scoreRange(letters, wordCount, from, to) {
  const marks = new Uint8Array(WORD_LENGTH);
  const unclaimed = new Uint8Array(ALPHABET);
  const buckets = new Int32Array(PATTERN_COUNT);
  const touched = new Int32Array(PATTERN_COUNT);
  const invN = 1 / wordCount;
  const results = [];

  for (let g = from; g < to; g++) {
    const gOff = g * WORD_LENGTH;
    let touchedCount = 0;

    for (let a = 0; a < wordCount; a++) {
      const aOff = a * WORD_LENGTH;

      for (let i = 0; i < WORD_LENGTH; i++) {
        const gl = letters[gOff + i];
        const al = letters[aOff + i];
        if (gl === al) {
          marks[i] = 2;
        } else {
          marks[i] = 0;
          unclaimed[al]++;
        }
      }
      for (let i = 0; i < WORD_LENGTH; i++) {
        if (marks[i] === 2) continue;
        const gl = letters[gOff + i];
        if (unclaimed[gl] > 0) {
          marks[i] = 1;
          unclaimed[gl]--;
        }
      }
      for (let i = 0; i < WORD_LENGTH; i++) unclaimed[letters[aOff + i]] = 0;

      const code = marks[0] + marks[1] * 3 + marks[2] * 9 + marks[3] * 27 + marks[4] * 81;
      if (buckets[code] === 0) touched[touchedCount++] = code;
      buckets[code]++;
    }

    let bits = 0;
    for (let t = 0; t < touchedCount; t++) {
      const code = touched[t];
      const p = buckets[code] * invN;
      bits -= p * Math.log2(p);
      buckets[code] = 0;
    }
    results.push({ index: g, bits });
  }
  return results;
}

if (!isMainThread) {
  const { letters, wordCount, from, to } = workerData;
  parentPort.postMessage(scoreRange(letters, wordCount, from, to));
} else {
  const topFlag = process.argv.indexOf('--top');
  const top = topFlag === -1 ? 20 : Number(process.argv[topFlag + 1]);
  if (!Number.isInteger(top) || top < 1) {
    console.error('--top must be a positive integer');
    process.exit(1);
  }

  const words = loadWords();
  const letters = pack(words);
  const n = words.length;
  const threads = Math.min(availableParallelism(), 16);
  const started = Date.now();

  console.log(`Scoring ${n} guesses against ${n} candidates (${n * n} evaluations) on ${threads} threads...`);

  const chunk = Math.ceil(n / threads);
  const jobs = [];
  for (let t = 0; t < threads; t++) {
    const from = t * chunk;
    const to = Math.min(from + chunk, n);
    if (from >= to) break;
    jobs.push(
      new Promise((resolvePromise, rejectPromise) => {
        const worker = new Worker(fileURLToPath(import.meta.url), {
          workerData: { letters, wordCount: n, from, to },
        });
        worker.on('message', resolvePromise);
        worker.on('error', rejectPromise);
        worker.on('exit', (code) => {
          if (code !== 0) rejectPromise(new Error(`worker exited with code ${code}`));
        });
      }),
    );
  }

  const scored = (await Promise.all(jobs)).flat();
  // Highest information first; alphabetical order breaks ties so the output is
  // reproducible.
  scored.sort((a, b) => b.bits - a.bits || (words[a.index] < words[b.index] ? -1 : 1));

  const openers = scored.slice(0, top).map(({ index, bits }) => ({
    word: words[index],
    bits: Number(bits.toFixed(6)),
  }));

  writeFileSync(
    TARGET,
    `${JSON.stringify(
      {
        note:
          'Generated by scripts/gen-openers.mjs. DO NOT EDIT. Expected information gain, in bits, ' +
          'for each opening guess scored against the full word list.',
        wordCount: n,
        openers,
      },
      null,
      2,
    )}\n`,
  );

  const elapsed = ((Date.now() - started) / 1000).toFixed(1);
  console.log(`Wrote src/data/openers.json in ${elapsed}s. Best openers:`);
  for (const { word, bits } of openers.slice(0, 10)) {
    console.log(`  ${word}  ${bits.toFixed(4)} bits`);
  }
}
