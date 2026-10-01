package main

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// humanLimit is the most runes a For Human bullet may carry: one fact a
// reader takes in at a glance, not a summary of the memo below.
const humanLimit = 30

// checkTaskShape enforces the body shape every task is written in:
//
//	# <title>
//	## For Human      bullets only (plus ### headings), each ≤ humanLimit runes
//	---
//	## AI memo        free markdown
//
// It returns the first violation, with a 1-based line number.
func checkTaskShape(body []byte) error {
	lines := strings.Split(strings.TrimRight(string(body), "\n"), "\n")
	human, rule, memo := 0, 0, 0
	for i, l := range lines {
		switch strings.TrimRight(l, " \t") {
		case "## For Human":
			if human == 0 {
				human = i + 1
			}
		case "---":
			if human > 0 && rule == 0 {
				rule = i + 1
			}
		case "## AI memo":
			if memo == 0 {
				memo = i + 1
			}
		}
	}
	switch {
	case human == 0:
		return fmt.Errorf("body has no \"## For Human\" section")
	case memo == 0:
		return fmt.Errorf("body has no \"## AI memo\" section")
	case memo < human:
		return fmt.Errorf("line %d: \"## AI memo\" must come after \"## For Human\"", memo)
	case rule == 0 || rule > memo:
		return fmt.Errorf("line %d: \"## AI memo\" must be separated from For Human by a --- line", memo)
	}
	for i := human; i < rule-1; i++ {
		l := strings.TrimRight(lines[i], " \t")
		t := strings.TrimLeft(l, " \t")
		switch {
		case t == "" || strings.HasPrefix(t, "### "):
			continue
		case strings.HasPrefix(t, "- "):
			text := strings.TrimPrefix(t, "- ")
			for _, box := range []string{"[ ] ", "[x] ", "[X] "} {
				text = strings.TrimPrefix(text, box)
			}
			if n := utf8.RuneCountInString(text); n > humanLimit {
				return fmt.Errorf("line %d: For Human bullet is %d characters, the limit is %d: %s", i+1, n, humanLimit, text)
			}
		default:
			return fmt.Errorf("line %d: For Human holds only bullets and ### headings: %s", i+1, l)
		}
	}
	return nil
}
