package data

import "testing"

func TestParseContextQuantitySupportsKAndM(t *testing.T) {
	cases := []struct {
		name      string
		input     any
		allowUnit bool
		want      int64
		wantOK    bool
	}{
		{name: "plain int", input: 300000, allowUnit: true, want: 300000, wantOK: true},
		{name: "plain string", input: "300000", allowUnit: true, want: 300000, wantOK: true},
		{name: "upper k", input: "300K", allowUnit: true, want: 300000, wantOK: true},
		{name: "decimal m", input: "0.8M", allowUnit: true, want: 800000, wantOK: true},
		{name: "spaced unit", input: "1.05 M", allowUnit: true, want: 1050000, wantOK: true},
		{name: "empty inherits", input: "", allowUnit: true, want: 0, wantOK: true},
		{name: "unit not allowed", input: "8K", allowUnit: false, wantOK: false},
		{name: "bad string", input: "bad", allowUnit: true, wantOK: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := parseContextQuantity(tc.input, tc.allowUnit)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if got != tc.want {
				t.Fatalf("value = %d, want %d", got, tc.want)
			}
		})
	}
}
