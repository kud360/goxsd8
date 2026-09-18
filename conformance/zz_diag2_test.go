package conformance

import (
	"os"
	"testing"
)

// diagProbeCases are the suite instance cases whose governing complex type
// TestDiagContentMatcherDeclines found ContentMatcher declining through
// partitionsBounded. THROWAWAY diagnostic for issue #1588; delete before
// handoff.
var diagProbeCases = []string{
	"MS-Particles2006-07-15/particlesZ007/instance/particlesZ007.i",
	"MS-Particles2006-07-15/particlesZ034_a1/instance/particlesZ034_a1.v",
	"MS-Particles2006-07-15/particlesZ034_a2/instance/particlesZ034_a2.i",
	"MS-Particles2006-07-15/particlesZ034_a3/instance/particlesZ034_a3.i",
	"MS-Particles2006-07-15/particlesZ034_b/instance/particlesZ034_b.i",
	"MS-Particles2006-07-15/particlesZ035_a/instance/particlesZ035_a.i",
	"MS-Particles2006-07-15/particlesZ036_a/instance/particlesZ036_a.i",
	"MS-Particles2006-07-15/particlesZ036_b1/instance/particlesZ036_b1.i",
	"MS-Particles2006-07-15/particlesZ036_b2/instance/particlesZ036_b2.i",
	"MS-Particles2006-07-15/particlesZ036_c/instance/particlesZ036_c.v",
}

// TestDiagProbeDecliningCases runs the instance lane executor over
// diagProbeCases alone and reports each case's Status and whether the decline
// census probe calls it DECLINED (fail under both expectation polarities) or
// DECIDED. Run with DIAG=1.
func TestDiagProbeDecliningCases(t *testing.T) {
	if os.Getenv("DIAG") != "1" {
		t.Skip("DIAG=1 not set")
	}
	skipWithoutSuite(t)
	found, err := parseSuite(suitePath())
	if err != nil {
		t.Fatalf("parsing suite: %v", err)
	}
	wanted := map[string]struct{}{}
	for _, id := range diagProbeCases {
		wanted[id] = struct{}{}
	}
	var l lane
	for _, cand := range defaultLanes() {
		if cand.name == "instance" {
			l = cand
		}
	}
	for _, c := range found.cases {
		if _, ok := wanted[c.id]; !ok {
			continue
		}
		st := l.exec(c)
		flipped := l.exec(flipExpectation(c))
		verdict := "DECIDED"
		if !st.IsPass() && !flipped.IsPass() {
			verdict = "DECLINED"
		}
		t.Logf("%s: wantsValid=%v status.IsPass=%v flipped.IsPass=%v -> %s",
			c.id, c.expect.wantsValid(), st.IsPass(), flipped.IsPass(), verdict)
	}
}
