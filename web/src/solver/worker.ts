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

// The solver, off the main thread.
//
// Filtering is cheap, but ranking scores every word against every remaining
// candidate, which can run for a second or more. Doing that on the UI thread
// would freeze typing, so everything lives here and the page only ever handles
// messages.

import { ANSWER_COUNT, WORDS } from '../data/words';
import openersData from '../data/openers.json';
import { LIVE_RANK_LIMIT, rankGuesses } from './entropy';
import { onlyAnswers, suggest, suggestIndices, toWords } from './filter';
import { Knowledge } from './knowledge';
import type { RankedGuess, RankingSource, SolveRequest, SolveResponse } from './protocol';

const openers: RankedGuess[] = openersData.openers.map(({ word, bits }) => ({
  word,
  bits,
  // Before any guess every word is still a possible answer.
  candidate: true,
}));

function solve(request: SolveRequest): SolveResponse {
  const started = performance.now();
  const knowledge = new Knowledge();

  for (const [index, turn] of request.turns.entries()) {
    try {
      knowledge.apply(turn.guess, turn.pattern);
    } catch (error) {
      return {
        id: request.id,
        ok: false,
        turnIndex: index,
        message: error instanceof Error ? error.message : String(error),
      };
    }
  }

  // No turns yet means every word is a candidate, and the opening ranking is
  // the precomputed one.
  if (request.turns.length === 0) {
    return {
      id: request.id,
      ok: true,
      candidates: WORDS,
      likelyCandidates: ANSWER_COUNT,
      solved: null,
      ranked: openers.slice(0, request.limit),
      rankedCandidates: [],
      rankingSource: 'precomputed',
      elapsedMs: performance.now() - started,
    };
  }

  // A solved puzzle has to be recognised before anything is said about the
  // candidate list, which is empty here: the winning word is a played word, so
  // the filter drops it like any other. Reporting that as "no word fits this
  // feedback" told winners they had made a mistake.
  const solved = knowledge.solved;
  if (solved !== null) {
    return {
      id: request.id,
      ok: true,
      candidates: [],
      likelyCandidates: 0,
      solved,
      ranked: [],
      rankedCandidates: [],
      rankingSource: 'none',
      elapsedMs: performance.now() - started,
    };
  }

  const candidates = suggest(knowledge);
  if (candidates.length === 0) {
    return {
      id: request.id,
      ok: true,
      candidates,
      likelyCandidates: 0,
      solved: null,
      ranked: [],
      rankedCandidates: [],
      rankingSource: 'none',
      elapsedMs: performance.now() - started,
    };
  }

  const allIndices = suggestIndices(knowledge);
  const likelyIndices = onlyAnswers(allIndices);

  // Score against the words that could plausibly be the answer, not against
  // every legal guess. Five in six legal guesses have never been solutions, and
  // letting them vote drags the ranking toward separating words nobody will
  // ever have to distinguish. Every surviving word is still reported; only the
  // probability model narrows.
  //
  // When nothing plausible survives -- the answer is a word outside the known
  // list -- this falls back to the full candidate set rather than giving up.
  const distribution = likelyIndices.length > 0 ? likelyIndices : allIndices;

  let ranked: readonly RankedGuess[] = [];
  let rankedCandidates: readonly RankedGuess[] = [];
  let rankingSource: RankingSource = 'deferred';

  if (distribution.length <= LIVE_RANK_LIMIT || request.force) {
    rankingSource = 'live';

    // Words that could win outright, best first. Scoring the distribution
    // against itself is cheap next to the full sweep below.
    rankedCandidates = rankGuesses(distribution, {
      limit: request.limit,
      candidatesOnly: true,
    });

    // A probe that cannot be the answer earns its place only by being strictly
    // more informative than the best word that could win. Otherwise the honest
    // advice is "guess one of those, you might just win".
    //
    // Without this the list fills with noise: once one candidate remains every
    // guess scores zero bits, and the alphabetical tie-break happily
    // recommended AAHED and AALII as ways to narrow a field of one.
    const bestWinnableBits = rankedCandidates[0]?.bits ?? 0;
    ranked = rankGuesses(distribution, { limit: request.limit })
      .filter((guess) => !guess.candidate && guess.bits > bestWinnableBits + 1e-9)
      .slice(0, 5);
  }

  return {
    id: request.id,
    ok: true,
    // Every survivor, in dictionary order, so nothing is ever hidden.
    candidates: toWords(allIndices),
    likelyCandidates: likelyIndices.length,
    solved: null,
    ranked,
    rankedCandidates,
    rankingSource,
    elapsedMs: performance.now() - started,
  };
}

self.addEventListener('message', (event: MessageEvent<SolveRequest>) => {
  self.postMessage(solve(event.data));
});
