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

import { WORDS } from '../data/words';
import openersData from '../data/openers.json';
import { LIVE_RANK_LIMIT, rankGuesses } from './entropy';
import { suggest, suggestIndices } from './filter';
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
      ranked: openers.slice(0, request.limit),
      rankedCandidates: [],
      rankingSource: 'precomputed',
      elapsedMs: performance.now() - started,
    };
  }

  const candidates = suggest(knowledge);
  if (candidates.length === 0) {
    return {
      id: request.id,
      ok: true,
      candidates,
      ranked: [],
      rankedCandidates: [],
      rankingSource: 'none',
      elapsedMs: performance.now() - started,
    };
  }

  let ranked: readonly RankedGuess[] = [];
  let rankedCandidates: readonly RankedGuess[] = [];
  let rankingSource: RankingSource = 'deferred';

  if (candidates.length <= LIVE_RANK_LIMIT || request.force) {
    const indices = suggestIndices(knowledge);
    ranked = rankGuesses(indices, { limit: request.limit });
    rankingSource = 'live';

    // Scoring the candidates against themselves is O(n^2) on an already small
    // set, so this is cheap next to the full sweep just performed. Skipped when
    // the best overall guesses are already mostly winnable words, since then
    // the second list would just repeat the first.
    if (ranked.filter((guess) => guess.candidate).length < 3) {
      rankedCandidates = rankGuesses(indices, { limit: 5, candidatesOnly: true });
    }
  }

  return {
    id: request.id,
    ok: true,
    candidates,
    ranked,
    rankedCandidates,
    rankingSource,
    elapsedMs: performance.now() - started,
  };
}

self.addEventListener('message', (event: MessageEvent<SolveRequest>) => {
  self.postMessage(solve(event.data));
});
