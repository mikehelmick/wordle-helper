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

import './styles.css';

import type { SolveRequest, SolveResponse, Turn } from './solver/protocol';
import { Board } from './ui/board';
import { Results } from './ui/results';

const RANKED_LIMIT = 10;
const THEME_KEY = 'wordle-helper:theme';

function required<T extends Element>(id: string): T {
  const element = document.getElementById(id);
  if (!element) throw new Error(`missing element #${id}`);
  return element as unknown as T;
}

const elements = {
  board: required<HTMLElement>('board'),
  status: required<HTMLElement>('status'),
  submit: required<HTMLButtonElement>('submit'),
  undo: required<HTMLButtonElement>('undo'),
  reset: required<HTMLButtonElement>('reset'),
  themeToggle: required<HTMLButtonElement>('theme-toggle'),
  themeIcon: required<HTMLElement>('theme-icon'),
};

// ---------------------------------------------------------------------------
// Theme

function applyTheme(theme: 'dark' | 'light'): void {
  document.documentElement.dataset['theme'] = theme;
  elements.themeToggle.setAttribute('aria-pressed', String(theme === 'light'));
  elements.themeIcon.textContent = theme === 'light' ? '☀' : '☾';
}

// Dark is the default; a stored choice wins. Reading localStorage can throw
// when storage is blocked, and a theme preference is not worth failing over.
let theme: 'dark' | 'light' = 'dark';
try {
  if (localStorage.getItem(THEME_KEY) === 'light') theme = 'light';
} catch {
  /* storage unavailable */
}
applyTheme(theme);

elements.themeToggle.addEventListener('click', () => {
  theme = theme === 'dark' ? 'light' : 'dark';
  applyTheme(theme);
  try {
    localStorage.setItem(THEME_KEY, theme);
  } catch {
    /* storage unavailable */
  }
});

// ---------------------------------------------------------------------------
// Solver worker

const worker = new Worker(new URL('./solver/worker.ts', import.meta.url), { type: 'module' });

/**
 * Only the newest request's reply is used. Ranking can take a moment, and a
 * fast second edit must not be overwritten by the first request finishing late.
 */
let latestRequestId = 0;
let lastTurns: Turn[] = [];

function solve(turns: Turn[], force = false): void {
  lastTurns = turns;
  latestRequestId += 1;
  const request: SolveRequest = { id: latestRequestId, turns, limit: RANKED_LIMIT, force };
  results.showPending();
  worker.postMessage(request);
}

// ---------------------------------------------------------------------------
// UI

const results = new Results(
  {
    count: required<HTMLElement>('remaining-count'),
    label: required<HTMLElement>('remaining-label'),
    ranked: required<HTMLElement>('ranked'),
    allWords: required<HTMLDetailsElement>('all-words'),
    allWordsCount: required<HTMLElement>('all-words-count'),
    allWordsList: required<HTMLElement>('all-words-list'),
  },
  { onForceRank: () => solve(lastTurns, true) },
);

function setStatus(text: string, tone: 'info' | 'error'): void {
  elements.status.textContent = text;
  elements.status.dataset['tone'] = tone;
}

const board = new Board(elements.board, {
  onTurnsChanged: solve,
  onMessage: setStatus,
  onStateChanged(state) {
    elements.submit.disabled = !state.canSubmit;
    elements.undo.disabled = !state.canUndo;
    elements.reset.disabled = !state.canReset;
  },
});

elements.submit.addEventListener('click', () => board.submit());
elements.undo.addEventListener('click', () => board.undo());
elements.reset.addEventListener('click', () => board.reset());

worker.addEventListener('message', (event: MessageEvent<SolveResponse>) => {
  const response = event.data;
  // Discard replies to superseded requests.
  if (response.id !== latestRequestId) return;

  if (!response.ok) {
    board.rejectTurn(response.turnIndex, `That does not fit the earlier rows: ${response.message}.`);
    return;
  }

  results.render(response);

  if (response.candidates.length === 1) {
    setStatus(`Only ${response.candidates[0]} fits. That is the answer.`, 'info');
  }
});

worker.addEventListener('error', (event) => {
  setStatus(`The solver failed to start: ${event.message}`, 'error');
});

// Prime the display with the opening state.
solve([]);
setStatus('Type your first guess, then colour the tiles to match.', 'info');
