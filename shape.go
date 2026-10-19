package main

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// lineLimit is the most runes a line under the title may carry: one fact a
// reader takes in at a glance.
const lineLimit = 30

// sections are the headings every task carries, in this order: SCQA, the
// introduction Barbara Minto lays out in The Pyramid Principle.
var sections = []string{"## Situation", "## Complication", "## Question", "## Answer"}

// whyHeading is the section a dropped task must also carry: the reason the
// work was decided against.
const whyHeading = "## Why dropped"

// shapeHelp is the task shape as the help states it; every other command
// points here (see task validate) instead of restating it.
var shapeHelp = fmt.Sprintf("the task shape is \"# <title>\", then %q, %q, %q, %q in that order and no other \"## \" heading; every line under the title is at most %d characters.\na task in dropped/ also carries %q with at least one line under it.", sections[0], sections[1], sections[2], sections[3], lineLimit, whyHeading)

// checkTaskShape enforces the body shape every task is written in:
//
//	# <title>
//	## Situation      what is so now
//	## Complication   what goes wrong with it
//	## Question       what the task has to answer
//	## Answer         the answer, once there is one
//
// Every line under the title is at most lineLimit runes, and no other "## "
// heading is allowed, so the body stays what the user reads. A task in
// dropped/ must also carry whyHeading with at least one line under it. It
// returns the first violation, with a 1-based line number.
func checkTaskShape(status string, body []byte) error {
	lines := strings.Split(strings.TrimRight(string(body), "\n"), "\n")
	next, why := 0, 0
	for i, l := range lines {
		l = strings.TrimRight(l, " \t")
		if !strings.HasPrefix(l, "## ") {
			if n := utf8.RuneCountInString(l); next > 0 && n > lineLimit {
				return fmt.Errorf("line %d: line is %d characters, the limit is %d: %s", i+1, n, lineLimit, l)
			}
			continue
		}
		switch {
		case next < len(sections) && l == sections[next]:
			next++
		case l == whyHeading && why == 0:
			why = i + 1
		default:
			if next < len(sections) {
				return fmt.Errorf("line %d: expected %q, got %q", i+1, sections[next], l)
			}
			return fmt.Errorf("line %d: %q is not a task section", i+1, l)
		}
	}
	if next < len(sections) {
		return fmt.Errorf("body has no %q section", sections[next])
	}
	if status == "dropped" {
		if why == 0 {
			return fmt.Errorf("a dropped task needs a %q section", whyHeading)
		}
		reason := false
		for i := why; i < len(lines) && !reason; i++ {
			l := strings.TrimSpace(lines[i])
			if strings.HasPrefix(l, "## ") {
				break
			}
			reason = l != "" && !strings.HasPrefix(l, "#")
		}
		if !reason {
			return fmt.Errorf("line %d: %q has no line under it", why, whyHeading)
		}
	}
	return nil
}
