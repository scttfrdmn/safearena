package main

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestSafeArenaPatterns runs the arenacheck analyzer against the satest package
// in testdata/src and verifies that unsafe patterns are flagged and safe patterns are not.
func TestSafeArenaPatterns(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, AnalyzerFinal2, "satest")
}
