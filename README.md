# wordle-helper

A helper to cheat... er... solve the daily Wordle. You tell it what you guessed
and what colours came back; it tells you which words are still possible.

It comes in two forms that share one word list and one algorithm:

- a **command line tool**, written in Go
- a **web page** that runs entirely in your browser, with no install and no
  network calls

## Use it in the browser

<https://mikehelmick.github.io/wordle-helper/>

Type the word you played, click each tile to cycle it through grey, yellow and
green to match the game, and press <kbd>Enter</kbd>. The remaining possibilities
update as you go, ranked by how much each guess is expected to narrow the field.

## Use it from the command line

You need [Go](https://go.dev/doc/install) 1.25 or newer. The module has no
dependencies.

```sh
go install github.com/mikehelmick/wordle-helper@latest
wordle-helper
```

Enter each guess, then its result pattern, using `.` for a grey tile, `Y` for
yellow and `G` for green.

```
$ wordle-helper
Wordle helper - I'm only interested in wrong guesses
Type "EXIT" to quit

Wrong guess 1: CRANE
Pattern (. = not in word, Y = wrong spot, G = correct): ..G..
Found 323 suggestions
Display all 323 suggestions [y/N]: n

Wrong guess 2: ABAFT
Pattern (. = not in word, Y = wrong spot, G = correct): ..G.G
Found 15 suggestions
GHAST GHAUT HOAST LOAST PLAIT PLAST PLATT SHAKT SHALT SKATT SMALT SPAIT SPALT SWAPT TOAST

Wrong guess 3: GHAST
Pattern (. = not in word, Y = wrong spot, G = correct): ..GGG
Found 3 suggestions
LOAST PLAST TOAST
```

Mistyped entries are re-asked rather than costing a round, and feedback that
contradicts an earlier turn is reported instead of silently producing nonsense.
Since input is read a line at a time, a whole session can also be piped in:

```sh
printf 'CRANE\n..G..\nn\nABAFT\n..G.G\nGHAST\n..GGG\nEXIT\n' | wordle-helper
```

## How it works

The solver keeps a running picture of the answer and filters the dictionary
against it. For each turn it records:

- which letter is fixed at each position (green)
- which letters cannot be at a given position (yellow, and grey letters that
  appear elsewhere in the same guess)
- how many times each letter may appear, as a minimum *and* a maximum

That last point is what makes repeated letters work. A grey tile does not simply
mean "this letter is absent": if the same letter is green or yellow elsewhere in
the guess, the grey one proves an exact count. Guessing `SASSY` against `MOSSY`
therefore teaches "exactly two S", not merely "at least one S".

The knowledge accumulates across turns, so a letter confirmed in round one keeps
constraining round three.

### Two word lists

Wordle accepts far more words than it will ever use as an answer. This ships
both, because the difference matters:

| File | Size | What it is |
| --- | --- | --- |
| [`pkg/wordle/words5.txt`](pkg/wordle/words5.txt) | 12,972 | every word the game accepts as a guess |
| [`pkg/wordle/answers5.txt`](pkg/wordle/answers5.txt) | 2,315 | words that have ever been the solution |

Five legal guesses in six have never been answers. `LOAST` and `PLAST` are
perfectly legal to type, but nobody has ever had to guess them. Reporting them
as though they were live possibilities buries the word you actually want, so
they are **ranked below the plausible words, not filtered out** — the puzzle's
answers are editorially chosen now rather than drawn from a fixed list, so a
word outside the answer file can still turn up, and hiding it would be worse
than cluttering the list. The CLI reports both counts and prints a `Likely:`
line; the web version leads with the plausible words and shows the rest behind a
disclosure.

The answer file is sorted alphabetically rather than in puzzle order, which is
deliberate: the original list ships chronologically, and committing it in that
order would leak every future answer.

### Ranking

The web version scores guesses by expected information gain. For a candidate
guess, partition the remaining possible answers by the feedback that guess would
produce, and score it by the entropy of that partition — the bits you expect to
learn. A guess that splits the field into many small groups beats one that
leaves a single large group.

Scoring uses the answer list as its probability model, since optimising for
words that can never come up is wasted effort. A word that cannot itself be the
answer is still offered when it genuinely separates the field faster, but only
then: if no probe beats simply guessing a word that could win, the tool says so
by not offering one.

The opening guess is precomputed by `web/scripts/gen-openers.mjs`, because
scoring all 12,972 guesses against every possible answer is 30 million
evaluations. It puts `SOARE` first at 5.89 bits.

## Development

```sh
# Go
go build ./...
go test ./... -race -cover
go vet ./... && staticcheck ./... && govulncheck ./...

# Web
cd web
npm install
npm run dev        # local dev server
npm test           # includes the cross-implementation checks below
npm run build
```

Two extra command line tools are useful when poking at the solver:

```sh
go run ./cmd/tester TOAST   # play a whole game against a known answer
go run ./cmd/clusters 8     # words differing in one position, e.g. _IGHT
```

### Keeping the two implementations honest

The same algorithm exists in Go and in TypeScript, which is an invitation to
drift. Two things prevent it:

- **The word list has one source.** `web/scripts/gen-words.mjs` generates
  `web/src/data/words.ts` from `pkg/wordle/words5.txt`. CI regenerates it and
  fails if the committed copy is stale.
- **The behaviour has one set of expectations.**
  `pkg/wordle/testdata/vectors.json` is generated from the Go implementation and
  asserted against by *both* test suites. If they ever disagree, both fail.

After a deliberate behaviour change, regenerate and re-run:

```sh
go test ./pkg/wordle -run TestVectors -update
cd web && npm test
```

Regenerate the precomputed openers only if the word list changes:

```sh
cd web && npm run gen:openers
```

## Layout

```
main.go                       the interactive CLI
pkg/wordle/                   the solver: dictionary, knowledge, feedback, filtering
pkg/wordle/words5.txt         the word list, embedded at build time
pkg/wordle/testdata/          shared Go/TypeScript test vectors
cmd/tester/                   plays a game against a known answer
cmd/clusters/                 finds words that differ in exactly one position
web/                          the browser front end (Vite + TypeScript)
```

## License

Apache 2.0. See [LICENSE](LICENSE).
