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

import type { PendingMark } from './board';

/** The three letter rows, in the familiar QWERTY arrangement. */
const LETTER_ROWS = ['QWERTYUIOP', 'ASDFGHJKL', 'ZXCVBNM'] as const;

/** The colours the shift keys arm, with the label each key carries. */
const SHIFT_KEYS: ReadonlyArray<{ mark: 'present' | 'correct'; label: string }> = [
  { mark: 'present', label: 'Wrong spot' },
  { mark: 'correct', label: 'Correct' },
];

export interface KeyboardOptions {
  /** A letter key was pressed. */
  onKey(letter: string): void;
  /** The Enter key was pressed. */
  onEnter(): void;
  /** The Backspace key was pressed. */
  onBackspace(): void;
  /** A colour shift key was pressed; the board decides whether to arm or cancel. */
  onArm(mark: 'present' | 'correct'): void;
}

/**
 * The on-screen keyboard. It exists so the app is usable on a phone, where there
 * is no physical keyboard to type a guess.
 *
 * Its twist is that the yellow and green markers are *shift* keys: pressing one
 * tints the whole keyboard and arms that colour, so the next letter is entered
 * already coloured. The board owns the armed colour; this component only reports
 * presses upward and reflects the armed colour back down via {@link setPending}.
 */
export class Keyboard {
  private readonly shiftButtons = new Map<'present' | 'correct', HTMLButtonElement>();

  constructor(
    private readonly root: HTMLElement,
    private readonly options: KeyboardOptions,
  ) {
    this.root.classList.add('keyboard');

    this.root.append(this.buildLetterRow(LETTER_ROWS[0]));
    this.root.append(this.buildLetterRow(LETTER_ROWS[1]));
    this.root.append(this.buildBottomRow(LETTER_ROWS[2]));
    this.root.append(this.buildShiftRow());
  }

  /**
   * Reflects the board's armed colour: tints the letter keys via a data
   * attribute the stylesheet keys off, and lights the matching shift key.
   */
  setPending(mark: PendingMark): void {
    if (mark) {
      this.root.dataset['arm'] = mark;
    } else {
      delete this.root.dataset['arm'];
    }
    for (const [keyMark, button] of this.shiftButtons) {
      button.setAttribute('aria-pressed', String(keyMark === mark));
    }
  }

  private buildLetterRow(letters: string): HTMLDivElement {
    const row = this.buildRow();
    for (const letter of letters) {
      row.append(
        this.buildKey(letter, 'key key--letter', () => this.options.onKey(letter)),
      );
    }
    return row;
  }

  /** The Z-row, flanked by Enter and Backspace as in the game's own keyboard. */
  private buildBottomRow(letters: string): HTMLDivElement {
    const row = this.buildRow();
    row.append(this.buildKey('Enter', 'key key--wide', () => this.options.onEnter()));
    for (const letter of letters) {
      row.append(
        this.buildKey(letter, 'key key--letter', () => this.options.onKey(letter)),
      );
    }
    // A wide backspace glyph; the accessible name says what it does.
    const back = this.buildKey('⌫', 'key key--wide', () => this.options.onBackspace());
    back.setAttribute('aria-label', 'Backspace');
    row.append(back);
    return row;
  }

  private buildShiftRow(): HTMLDivElement {
    const row = this.buildRow();
    for (const { mark, label } of SHIFT_KEYS) {
      row.append(this.buildShiftKey(mark, label));
    }
    return row;
  }

  private buildShiftKey(mark: 'present' | 'correct', label: string): HTMLButtonElement {
    const button = this.buildKey(label, `key key--shift key--shift-${mark}`, () =>
      this.options.onArm(mark),
    );
    button.setAttribute('aria-pressed', 'false');

    // A swatch makes the colour legible next to the label, matching the legend.
    const swatch = document.createElement('i');
    swatch.className = 'key__swatch';
    swatch.setAttribute('aria-hidden', 'true');
    button.prepend(swatch);

    this.shiftButtons.set(mark, button);
    return button;
  }

  private buildRow(): HTMLDivElement {
    const row = document.createElement('div');
    row.className = 'keyboard__row';
    return row;
  }

  private buildKey(label: string, className: string, onClick: () => void): HTMLButtonElement {
    const button = document.createElement('button');
    button.type = 'button';
    button.className = className;
    button.append(document.createTextNode(label));
    button.addEventListener('click', onClick);
    // Keep focus on the body so a following physical Space still cycles the
    // active tile's colour rather than re-activating this button. Tab users who
    // deliberately focus a key still get normal Enter/Space activation.
    button.addEventListener('mousedown', (event) => event.preventDefault());
    return button;
  }
}
