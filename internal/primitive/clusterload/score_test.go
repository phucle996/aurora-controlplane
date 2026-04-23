package clusterload

import "testing"

func TestNormalizedScore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		load     int
		capacity int
		want     float64
	}{
		{name: "normal", load: 3, capacity: 4, want: 1},
		{name: "zero capacity treated as one", load: 2, capacity: 0, want: 3},
		{name: "negative capacity treated as one", load: 1, capacity: -5, want: 2},
		{name: "negative load clamped to zero", load: -10, capacity: 5, want: 0.2},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := NormalizedScore(tc.load, tc.capacity)
			if got != tc.want {
				t.Fatalf("expected score %v, got %v", tc.want, got)
			}
		})
	}
}
