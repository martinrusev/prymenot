package main

import (
	"fmt"
	"testing"
)

func TestParseLine(t *testing.T) {
	testCases := []struct {
		source string
		want   string
	}{
		{"127.0.0.1 005.free-counter.co.uk", "005.free-counter.co.uk"},  // separated by space
		{"127.0.0.1\t005.free-counter.co.uk", "005.free-counter.co.uk"}, // separated by tab
		{"0.0.0.0 0000mps.webpreview.dsl.net", "0000mps.webpreview.dsl.net"},
		{"0.0.0.0 http://ads1.msn.com", "ads1.msn.com"}, // cleanup https://
		{"beatyhousesupporte.su", "beatyhousesupporte.su"},
		{"127.0.0.1	localhost", ""}, // ignore localhost
		{"#0.0.0.0 anetno.tradedoubler.com", ""}, // ignore commented entries
	}
	for _, tc := range testCases {
		line, err := parseLine(tc.source)
		if err != nil {
			t.Fatalf("could not parse line location %q", tc.source)
		}
		fmt.Println(line)
		if line != tc.want {
			t.Errorf("Line: %s | Got: %s | Want: %s", tc.source, line, tc.want)
		}

	}
}
