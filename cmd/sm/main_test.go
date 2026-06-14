package main

import "testing"

func strPtr(s string) *string { return &s }

func eqStrPtr(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// TestScanDelayLeakedTokens covers the orderings that urfave/cli mishandles
// when a negative offset halts flag parsing: flags placed after the offset leak
// into the positional args and must be recovered.
func TestScanDelayLeakedTokens(t *testing.T) {
	tests := []struct {
		name    string
		rawArgs []string
		base    delayArgs
		want    delayArgs
	}{
		{
			name:    "negative offset with dry-run leaked after",
			rawArgs: []string{"file.srt", "-500", "--dry-run"},
			want:    delayArgs{file: "file.srt", offset: "-500", dryRun: true},
		},
		{
			name:    "positive offset, no leaked flags",
			rawArgs: []string{"file.srt", "+500"},
			want:    delayArgs{file: "file.srt", offset: "+500"},
		},
		{
			name:    "flags parsed before offset arrive via base",
			rawArgs: []string{"file.srt", "-500"},
			base:    delayArgs{dryRun: true, rangeVal: strPtr("3-4")},
			want:    delayArgs{file: "file.srt", offset: "-500", dryRun: true, rangeVal: strPtr("3-4")},
		},
		{
			name:    "leaked --range with separate value",
			rawArgs: []string{"file.srt", "-500", "--range", "3-4", "--dry-run"},
			want:    delayArgs{file: "file.srt", offset: "-500", rangeVal: strPtr("3-4"), dryRun: true},
		},
		{
			name:    "leaked --range=value and --from-timestamp=value",
			rawArgs: []string{"file.srt", "-500", "--range=2-9", "--from-timestamp=00:00:09,000"},
			want:    delayArgs{file: "file.srt", offset: "-500", rangeVal: strPtr("2-9"), fromTs: strPtr("00:00:09,000")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scanDelayLeakedTokens(tt.rawArgs, tt.base)
			if got.file != tt.want.file || got.offset != tt.want.offset || got.dryRun != tt.want.dryRun {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
			if !eqStrPtr(got.rangeVal, tt.want.rangeVal) {
				t.Fatalf("rangeVal: got %v, want %v", got.rangeVal, tt.want.rangeVal)
			}
			if !eqStrPtr(got.fromTs, tt.want.fromTs) {
				t.Fatalf("fromTs: got %v, want %v", got.fromTs, tt.want.fromTs)
			}
		})
	}
}
