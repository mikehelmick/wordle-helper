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

// Messages exchanged with the solver worker.
//
// This module holds types only, so the main bundle can import it without
// pulling in the 68 KB word list -- that lives in the worker, which is the only
// thing that needs it.

/** One recorded turn: the word played and the colours it came back as. */
export interface Turn {
  readonly guess: string;
  readonly pattern: string;
}

export interface SolveRequest {
  /** Correlates a response with its request; stale replies are discarded. */
  readonly id: number;
  readonly turns: readonly Turn[];
  /** How many ranked suggestions to return. */
  readonly limit: number;
  /** Rank even when the candidate set is large enough to be slow. */
  readonly force: boolean;
}

export interface RankedGuess {
  readonly word: string;
  readonly bits: number;
  readonly candidate: boolean;
}

/**
 * Where the ranking came from.
 *
 * `precomputed` is the opening guess, scored offline because the full sweep is
 * 168 million evaluations. `deferred` means the candidate set is still large
 * enough that ranking would block noticeably, so the user is offered the choice
 * rather than made to wait.
 */
export type RankingSource = 'precomputed' | 'live' | 'deferred' | 'none';

export interface SolveSuccess {
  readonly id: number;
  readonly ok: true;
  readonly candidates: readonly string[];
  /** Best guesses overall, which may well be words that cannot be the answer. */
  readonly ranked: readonly RankedGuess[];
  /**
   * Best guesses drawn only from the words that could actually win.
   *
   * The overall ranking optimises for information, so once the field is small
   * it fills up with probe words and the possible answers vanish from view.
   * Someone using this still wants to know what they might type to win, so the
   * two lists are reported separately.
   */
  readonly rankedCandidates: readonly RankedGuess[];
  readonly rankingSource: RankingSource;
  readonly elapsedMs: number;
}

export interface SolveFailure {
  readonly id: number;
  readonly ok: false;
  /** Index into `turns` of the entry that could not be applied. */
  readonly turnIndex: number;
  readonly message: string;
}

export type SolveResponse = SolveSuccess | SolveFailure;
