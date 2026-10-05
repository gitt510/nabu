package main

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// humanLimit is the most runes a For Human line may carry: one fact a
// reader takes in at a glance, not a summary of the memo below.
const humanLimit = 30

// whyHeading is the For Human heading a dropped task must carry: the
// reason the work was decided against, where the user reads state.
const whyHeading = "### Why dropped"

// checkTaskShape enforces the body shape every task is written in:
//
//	# <title>
//	## For Human      every line at most humanLimit runes
//	---
//	## AI memo        free markdown
//
// A task in dropped/ must also carry whyHeading inside For Human with at
// least one line under it. It returns the first violation, with a 1-based
// line number.
func checkTaskShape(status string, body []byte) error {
	lines := strings.Split(strings.TrimRight(string(body), "\n"), "\n")
	human, rule, memo, why := 0, 0, 0, 0
	for i, l := range lines {
		switch strings.TrimRight(l, " \t") {
		case "## For Human":
			if human == 0 {
				human = i + 1
			}
		case whyHeading:
			if human > 0 && rule == 0 && why == 0 {
				why = i + 1
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
		if n := utf8.RuneCountInString(l); n > humanLimit {
			return fmt.Errorf("line %d: For Human line is %d characters, the limit is %d: %s", i+1, n, humanLimit, l)
		}
	}
	if status == "dropped" {
		if why == 0 {
			return fmt.Errorf("a dropped task needs a %q heading under For Human", whyHeading)
		}
		reason := false
		for i := why; i < rule-1 && !reason; i++ {
			l := strings.TrimSpace(lines[i])
			reason = l != "" && !strings.HasPrefix(l, "#")
		}
		if !reason {
			return fmt.Errorf("line %d: %q has no line under it", why, whyHeading)
		}
	}
	return nil
}
