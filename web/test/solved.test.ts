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

import { describe, expect, it } from 'vitest';

import { suggest } from '../src/solver/filter';
import { Knowledge } from '../src/solver/knowledge';

// An all-green turn empties the candidate list: the winning word has been
// played, so the filter drops it like any other. That makes a solved puzzle
// indistinguishable from feedback that cannot be satisfied, and the page used
// to greet winners with "No word fits this feedback. Check the colours you
// entered — one of the rows is likely off."

function knowing(...turns: Array<[string, string]>): Knowledge {
  const k = new Knowledge();
  for (const [guess, pattern] of turns) k.apply(guess, pattern);
  return k;
}

describe('a solved puzzle', () => {
  it('is not reported as unsatisfiable feedback', () => {
    const k = knowing(['TOAST', 'GGGGG']);

    // The empty list is real, and is exactly why `solved` has to be tracked
    // separately rather than inferred from the count.
    expect(suggest(k)).toEqual([]);
    expect(k.solved).toBe('TOAST');
  });

  it('records the answer after several turns', () => {
    const k = knowing(['CRANE', '..G..'], ['ABAFT', '..G.G'], ['TOAST', 'GGGGG']);
    expect(k.solved).toBe('TOAST');
  });

  it('stays unsolved without a full row of greens', () => {
    for (const pattern of ['.....', 'GGGG.', 'GG.GG', 'YYYYY', 'GGGGY']) {
      expect(knowing(['TOAST', pattern]).solved, `pattern ${pattern}`).toBeNull();
    }
  });

  it('normalizes lowercase input', () => {
    expect(knowing(['toast', 'ggggg']).solved).toBe('TOAST');
  });

  it('is cleared by reset and carried by clone', () => {
    const k = knowing(['TOAST', 'GGGGG']);

    const copy = k.clone();
    expect(copy.solved).toBe('TOAST');

    k.reset();
    expect(k.solved).toBeNull();
    // The clone is independent, so it keeps its own record.
    expect(copy.solved).toBe('TOAST');
  });

  it('shows up in the debug description', () => {
    expect(knowing(['TOAST', 'GGGGG']).describe()).toContain('solved=TOAST');
    expect(knowing(['TOAST', '.....']).describe()).not.toContain('solved=');
  });

  it('distinguishes a win from genuinely impossible feedback', () => {
    // Both leave zero candidates. Only one of them is good news.
    //
    // Four all-grey guesses are needed to exhaust the dictionary: three still
    // leave HUDUD standing, which is a fair reminder that "no letters at all"
    // is a weaker constraint than it looks.
    const won = knowing(['TOAST', 'GGGGG']);
    const impossible = knowing(
      ['CRANE', '.....'],
      ['TOAST', '.....'],
      ['MILKY', '.....'],
      ['HUDUD', '.....'],
    );

    expect(suggest(won)).toEqual([]);
    expect(won.solved).toBe('TOAST');

    expect(suggest(impossible)).toEqual([]);
    expect(impossible.solved).toBeNull();
  });
});
