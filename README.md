# wordle-helper

A simple command line program to help you cheat... er... solve the daily wordle.

# Install

First, you need to have the Go programming language installed: https://go.dev/doc/install

1. Install the helper
   
```
go get github.com/mikehelmick/wordle-helper
```

1. Run the helper

```
wordle-helper
```

That's it!

# Example

```
$ wordle-helper
Wordle helper - I'm only interested in wrong guesses
Type 'EXIT' to quit
✔ Enter pattern . = not in word, Y = wrong spot, G = correct: : ..Y..█
Found 2141 suggestionst in word, Y = wrong spot, G = correct: 
✗ Display all 2141 suggestions?: 
Invalid guess 2: MAKES
Enter pattern . = not in word, Y = wrong spot, G = correct: : .Y...
Found 427 suggestions
✗ Display all 427 suggestions?: 
Invalid guess 3: BLOAT
Enter pattern . = not in word, Y = wrong spot, G = correct: : ..YGY
Found 3 suggestions
FOUAT TODAY TOPAZ
```