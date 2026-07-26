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

import type { RankedGuess, RankingSource, SolveSuccess } from '../solver/protocol';

/** How many words to spell out under "show every possibility". */
const ALL_WORDS_CAP = 2000;

/** At or below this many candidates, the full list is shown without a click. */
const AUTO_EXPAND_LIMIT = 30;

const numberFormat = new Intl.NumberFormat('en-US');

export interface ResultsElements {
  readonly count: HTMLElement;
  readonly label: HTMLElement;
  readonly ranked: HTMLElement;
  readonly allWords: HTMLDetailsElement;
  readonly allWordsCount: HTMLElement;
  readonly allWordsList: HTMLElement;
}

export interface ResultsOptions {
  /** Called when the user asks for a ranking that was deferred as too slow. */
  onForceRank(): void;
}

export class Results {
  constructor(
    private readonly elements: ResultsElements,
    private readonly options: ResultsOptions,
  ) {}

  showPending(): void {
    this.elements.label.textContent = 'working…';
  }

  render(result: SolveSuccess): void {
    const { candidates, ranked, rankedCandidates, rankingSource } = result;

    this.elements.count.textContent = numberFormat.format(candidates.length);
    this.elements.count.dataset['tone'] =
      candidates.length === 0 ? 'empty' : candidates.length === 1 ? 'solved' : 'normal';
    this.elements.label.textContent =
      candidates.length === 1 ? 'word possible — that is the answer' : 'words possible';

    this.renderRanked(ranked, rankedCandidates, rankingSource, candidates.length);
    this.renderAllWords(candidates);
  }

  private renderRanked(
    ranked: readonly RankedGuess[],
    rankedCandidates: readonly RankedGuess[],
    source: RankingSource,
    candidateCount: number,
  ): void {
    const container = this.elements.ranked;
    container.replaceChildren();

    if (candidateCount === 0) {
      container.append(
        note('No word fits this feedback. Check the colours you entered — one of the rows is likely off.'),
      );
      return;
    }

    if (source === 'deferred') {
      container.append(
        note(
          `Ranking ${numberFormat.format(candidateCount)} candidates takes a few seconds. ` +
            'Narrow it down with another guess, or rank them now.',
        ),
      );
      const button = document.createElement('button');
      button.type = 'button';
      button.className = 'button button--primary';
      button.textContent = 'Rank anyway';
      button.addEventListener('click', () => this.options.onForceRank(), { once: true });
      container.append(button);
      return;
    }

    if (ranked.length === 0) return;

    // When the field is small the best-information list fills up with probe
    // words, so the words that could actually win get their own list above it.
    if (rankedCandidates.length > 0) {
      container.append(
        note(
          candidateCount <= rankedCandidates.length
            ? 'Words that could be the answer, best first.'
            : `Best of the ${numberFormat.format(candidateCount)} words that could be the answer.`,
        ),
      );
      this.appendRows(container, rankedCandidates, source, { markCandidates: false });
      container.append(
        note('Best for narrowing the field, even if the word cannot itself be the answer.'),
      );
      this.appendRows(container, ranked, source, { markCandidates: true });
      return;
    }

    container.append(
      note(
        source === 'precomputed'
          ? 'Best openers, scored against the whole word list.'
          : 'Best next guesses, by how much each is expected to narrow the field.',
      ),
    );
    this.appendRows(container, ranked, source, { markCandidates: true });
  }

  private appendRows(
    container: HTMLElement,
    guesses: readonly RankedGuess[],
    source: RankingSource,
    options: { markCandidates: boolean },
  ): void {
    guesses.forEach((guess, index) => {
      const row = document.createElement('div');
      row.className = index === 0 ? 'ranked__row ranked__row--top' : 'ranked__row';

      const rank = document.createElement('span');
      rank.className = 'ranked__rank';
      rank.textContent = String(index + 1);

      const word = document.createElement('span');
      word.className = 'ranked__word';
      word.append(guess.word);
      if (options.markCandidates && guess.candidate && source !== 'precomputed') {
        const badge = document.createElement('span');
        badge.className = 'ranked__badge';
        badge.textContent = 'possible';
        badge.title = 'This word could itself be the answer';
        word.append(badge);
      }

      const bits = document.createElement('span');
      bits.className = 'ranked__bits';
      bits.textContent = `${guess.bits.toFixed(2)} bits`;
      bits.title = 'Expected information gain';

      row.append(rank, word, bits);
      container.append(row);
    });
  }

  private renderAllWords(candidates: readonly string[]): void {
    const { allWords, allWordsCount, allWordsList } = this.elements;

    allWordsCount.textContent = numberFormat.format(candidates.length);
    allWords.hidden = candidates.length === 0;
    // Once the list is short enough to read at a glance, hiding it behind a
    // disclosure just adds a click.
    allWords.open = candidates.length > 0 && candidates.length <= AUTO_EXPAND_LIMIT;

    // Painting 12,972 words costs more than it tells anyone, so the tail is
    // summarised rather than silently dropped.
    const shown = candidates.slice(0, ALL_WORDS_CAP);
    allWordsList.textContent =
      shown.join(' ') +
      (candidates.length > shown.length
        ? ` … and ${numberFormat.format(candidates.length - shown.length)} more`
        : '');
  }
}

function note(text: string): HTMLParagraphElement {
  const p = document.createElement('p');
  p.className = 'ranked__caption';
  p.textContent = text;
  return p;
}
