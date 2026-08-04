package xsd

// TEMPORARY instrumentation for issue #282 measurement. Delete before handoff.

import (
	"fmt"
	"os"
)

var (
	diagMaxVisited  int
	diagCeilingHits int
	diagWalks       int
)

func diagOut(format string, args ...any) {
	path := os.Getenv("GOXSD_DIAG282")
	if path == "" {
		return
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, format, args...)
}

func diagWalk() {
	diagWalks++
	if diagWalks%100 == 0 {
		diagOut("walks=%d maxVisited=%d ceilingHits=%d\n", diagWalks, diagMaxVisited, diagCeilingHits)
	}
}

func diagRecordVisited(n int) {
	if n <= diagMaxVisited {
		return
	}
	diagMaxVisited = n
	diagOut("high-water visited=%d (walk #%d)\n", n, diagWalks)
}

func diagCeilingHit(n int) {
	diagCeilingHits++
	diagOut("CEILING HIT #%d at visited=%d (walk #%d)\n", diagCeilingHits, n, diagWalks)
}
