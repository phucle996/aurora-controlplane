package clusterload

// NormalizedScore returns a normalized load score where lower is better.
// Capacity <= 0 is treated as 1 to avoid divide-by-zero and preserve safety.
func NormalizedScore(activeLoad int, capacity int) float64 {
	if capacity <= 0 {
		capacity = 1
	}
	if activeLoad < 0 {
		activeLoad = 0
	}
	return float64(activeLoad+1) / float64(capacity)
}
