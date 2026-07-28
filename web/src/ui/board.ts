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

import { ABSENT, CORRECT, PRESENT, WORD_LENGTH } from '../solver/errors';
import type { Turn } from '../solver/protocol';

const MAX_ROWS = 6;

/** Tile states, in the order clicking cycles through them. */
const MARKS = ['absent', 'present', 'correct'] as const;
type Mark = (typeof MARKS)[number];

const MARK_TO_CHAR: Record<Mark, string> = {
  absent: ABSENT,
  present: PRESENT,
  correct: CORRECT,
};

const MARK_LABEL: Record<Mark, string> = {
  absent: 'not in word',
  present: 'wrong spot',
  correct: 'correct',
};

interface Row {
  readonly element: HTMLDivElement;
  readonly tiles: HTMLButtonElement[];
  letters: string[];
  marks: Mark[];
  committed: boolean;
}

/** Which of the board's actions are currently available. */
export interface BoardState {
  readonly canSubmit: boolean;
  readonly canUndo: boolean;
  readonly canReset: boolean;
}

export interface BoardOptions {
  /** Fired whenever the set of committed turns changes. */
  onTurnsChanged(turns: Turn[]): void;
  /** Fired for anything worth telling the user, such as a rejected entry. */
  onMessage(text: string, tone: 'info' | 'error'): void;
  /**
   * Fired after every change, including typing a single letter, so controls
   * can enable and disable in step with the board.
   *
   * The state is passed rather than read back off the Board, because this fires
   * during construction, before the caller's own reference exists.
   */
  onStateChanged(state: BoardState): void;
}

/**
 * The six-by-five grid of guesses.
 *
 * Only the first uncommitted row accepts input. Letters are typed, colours are
 * cycled by clicking a tile or pressing Space, and Enter records the turn.
 */
export class Board {
  private readonly rows: Row[] = [];
  private cursor = 0;
  private active = 0;

  constructor(
    private readonly root: HTMLElement,
    private readonly options: BoardOptions,
  ) {
    for (let r = 0; r < MAX_ROWS; r++) {
      this.rows.push(this.buildRow(r));
    }
    this.render();

    document.addEventListener('keydown', this.handleKeyDown);
  }

  /** Turns recorded so far, oldest first. */
  turns(): Turn[] {
    return this.rows
      .filter((row) => row.committed)
      .map((row) => ({
        guess: row.letters.join(''),
        pattern: row.marks.map((mark) => MARK_TO_CHAR[mark]).join(''),
      }));
  }

  get committedCount(): number {
    return this.rows.filter((row) => row.committed).length;
  }

  get isEmpty(): boolean {
    return this.committedCount === 0 && this.rows[this.active]!.letters.length === 0;
  }

  /** Whether the row being edited is complete enough to record. */
  get canSubmit(): boolean {
    const row = this.rows[this.active];
    return row !== undefined && !row.committed && row.letters.length === WORD_LENGTH;
  }

  /** Records the row being edited, as pressing Enter does. */
  submit(): void {
    const row = this.rows[this.active];
    if (row && !row.committed) this.commit(row);
  }

  reset(): void {
    for (const row of this.rows) {
      row.letters = [];
      row.marks = [];
      row.committed = false;
    }
    this.active = 0;
    this.cursor = 0;
    this.render();
    this.options.onTurnsChanged(this.turns());
    this.options.onMessage('', 'info');
  }

  /** Removes the most recent committed turn and reopens it for editing. */
  undo(): void {
    const last = this.committedCount - 1;
    if (last < 0) return;

    // Clear whatever was being typed in the row below before reopening.
    const draft = this.rows[last + 1];
    if (draft) {
      draft.letters = [];
      draft.marks = [];
    }

    this.rows[last]!.committed = false;
    this.active = last;
    this.cursor = WORD_LENGTH;
    this.render();
    this.options.onTurnsChanged(this.turns());
    this.options.onMessage('Turn reopened. Adjust it and press Enter.', 'info');
  }

  /**
   * Rolls a turn back after the solver rejected it, which happens when the
   * colours contradict an earlier row.
   */
  rejectTurn(turnIndex: number, message: string): void {
    const row = this.rows[turnIndex];
    if (!row) return;

    for (let r = MAX_ROWS - 1; r >= turnIndex; r--) {
      this.rows[r]!.committed = false;
    }
    this.active = turnIndex;
    this.cursor = WORD_LENGTH;
    this.render();
    this.shake(row);
    this.options.onMessage(message, 'error');
    this.options.onTurnsChanged(this.turns());
  }

  private buildRow(index: number): Row {
    const element = document.createElement('div');
    element.className = 'board__row';

    const tiles: HTMLButtonElement[] = [];
    for (let c = 0; c < WORD_LENGTH; c++) {
      const tile = document.createElement('button');
      tile.type = 'button';
      tile.className = 'tile';
      tile.addEventListener('click', () => this.handleTileClick(index, c));
      tiles.push(tile);
      element.append(tile);
    }

    this.root.append(element);
    return { element, tiles, letters: [], marks: [], committed: false };
  }

  private handleTileClick(rowIndex: number, column: number): void {
    if (rowIndex !== this.active) return;
    const row = this.rows[rowIndex]!;
    if (column >= row.letters.length) {
      // Clicking past the last letter just moves the caret there.
      this.cursor = row.letters.length;
      this.render();
      return;
    }
    this.cursor = column;
    this.cycleMark(row, column);
  }

  private cycleMark(row: Row, column: number): void {
    const current = row.marks[column] ?? 'absent';
    const next = MARKS[(MARKS.indexOf(current) + 1) % MARKS.length]!;
    row.marks[column] = next;
    this.render();

    const tile = row.tiles[column]!;
    tile.animate?.(
      [{ transform: 'scale(1)' }, { transform: 'scale(1.08)' }, { transform: 'scale(1)' }],
      { duration: 130, easing: 'ease-out' },
    );
  }

  private readonly handleKeyDown = (event: KeyboardEvent): void => {
    if (event.metaKey || event.ctrlKey || event.altKey) return;

    const target = event.target;
    // Let the browser handle activating a real control such as Undo.
    if (
      target instanceof HTMLElement &&
      target.tagName === 'BUTTON' &&
      !target.classList.contains('tile') &&
      (event.key === 'Enter' || event.key === ' ')
    ) {
      return;
    }
    if (target instanceof HTMLElement && (target.tagName === 'SUMMARY' || target.tagName === 'A')) {
      return;
    }

    const row = this.rows[this.active];
    if (!row) return;

    if (/^[a-zA-Z]$/.test(event.key)) {
      event.preventDefault();
      this.typeLetter(row, event.key.toUpperCase());
      return;
    }

    switch (event.key) {
      case 'Backspace':
        event.preventDefault();
        this.deleteLetter(row);
        break;
      case 'Enter':
        event.preventDefault();
        this.commit(row);
        break;
      case ' ': {
        event.preventDefault();
        // The caret sits *past* the last letter, so on a full row `cursor`
        // equals the length and a naive bounds check rejected every press --
        // Space worked only mid-word, which is exactly when there is nothing to
        // colour. Clamp onto the last tile instead.
        const column = Math.min(this.cursor, row.letters.length - 1);
        if (column >= 0) {
          this.cursor = column;
          this.cycleMark(row, column);
        }
        break;
      }
      case 'ArrowLeft':
        event.preventDefault();
        this.cursor = Math.max(0, this.cursor - 1);
        this.render();
        break;
      case 'ArrowRight':
        event.preventDefault();
        this.cursor = Math.min(row.letters.length, this.cursor + 1);
        this.render();
        break;
      default:
        break;
    }
  };

  private typeLetter(row: Row, letter: string): void {
    if (this.cursor < row.letters.length) {
      // Overwrite in place, keeping whatever colour was already chosen.
      row.letters[this.cursor] = letter;
      this.cursor = Math.min(WORD_LENGTH - 1, this.cursor + 1);
      this.render();
      return;
    }
    if (row.letters.length >= WORD_LENGTH) return;

    row.letters.push(letter);
    row.marks.push('absent');
    this.cursor = row.letters.length;
    this.render();

    const tile = row.tiles[row.letters.length - 1]!;
    tile.dataset['animate'] = 'pop';
    tile.addEventListener('animationend', () => delete tile.dataset['animate'], { once: true });
  }

  private deleteLetter(row: Row): void {
    if (row.letters.length === 0) return;
    const at = this.cursor >= row.letters.length ? row.letters.length - 1 : this.cursor - 1;
    if (at < 0) return;

    row.letters.splice(at, 1);
    row.marks.splice(at, 1);
    this.cursor = at;
    this.render();
  }

  private commit(row: Row): void {
    if (row.letters.length < WORD_LENGTH) {
      this.shake(row);
      this.options.onMessage(`Enter all ${WORD_LENGTH} letters first.`, 'error');
      return;
    }

    row.committed = true;
    this.active = Math.min(this.active + 1, MAX_ROWS - 1);
    this.cursor = 0;
    this.render();

    row.tiles.forEach((tile, i) => {
      tile.dataset['animate'] = 'flip';
      tile.style.animationDelay = `${i * 90}ms`;
      tile.addEventListener(
        'animationend',
        () => {
          delete tile.dataset['animate'];
          tile.style.animationDelay = '';
        },
        { once: true },
      );
    });

    this.options.onMessage('', 'info');
    this.options.onTurnsChanged(this.turns());
  }

  private shake(row: Row): void {
    row.element.dataset['animate'] = 'shake';
    row.element.addEventListener('animationend', () => delete row.element.dataset['animate'], {
      once: true,
    });
  }

  private render(): void {
    this.rows.forEach((row, rowIndex) => {
      const isActive = rowIndex === this.active && !row.committed;
      row.element.dataset['state'] = row.committed ? 'committed' : isActive ? 'active' : 'idle';

      row.tiles.forEach((tile, column) => {
        const letter = row.letters[column] ?? '';
        const mark = row.marks[column];

        tile.textContent = letter;
        tile.dataset['filled'] = letter ? 'true' : 'false';
        if (letter && mark) {
          tile.dataset['mark'] = mark;
        } else {
          delete tile.dataset['mark'];
        }

        // The caret marks the tile the keyboard acts on. On a full row that is
        // the last tile rather than the (nonexistent) next empty one, and it has
        // to stay visible once the tile has a letter, or Space appears to do
        // nothing to nothing in particular.
        const caret = Math.min(this.cursor, WORD_LENGTH - 1);
        if (isActive && column === caret) {
          tile.dataset['cursor'] = 'true';
        } else {
          delete tile.dataset['cursor'];
        }

        const editable = isActive && letter !== '';
        tile.disabled = !isActive;
        tile.setAttribute(
          'aria-label',
          letter
            ? `Position ${column + 1}: ${letter}, ${MARK_LABEL[mark ?? 'absent']}${
                editable ? '. Activate to change.' : ''
              }`
            : `Position ${column + 1}: empty`,
        );
      });
    });

    this.options.onStateChanged({
      canSubmit: this.canSubmit,
      canUndo: this.committedCount > 0,
      canReset: !this.isEmpty,
    });
  }
}
