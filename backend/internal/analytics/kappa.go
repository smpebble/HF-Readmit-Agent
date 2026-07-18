package analytics

// CohenKappa returns unweighted Cohen's kappa for two equal-length sets of
// categorical ratings. It returns false when the metric is undefined.
func CohenKappa(left, right []string) (float64, bool) {
	if len(left) == 0 || len(left) != len(right) {
		return 0, false
	}
	leftCounts, rightCounts := map[string]int{}, map[string]int{}
	agree := 0
	for i := range left {
		leftCounts[left[i]]++
		rightCounts[right[i]]++
		if left[i] == right[i] {
			agree++
		}
	}
	n := float64(len(left))
	observed := float64(agree) / n
	expected := 0.0
	for tier, count := range leftCounts {
		expected += float64(count*rightCounts[tier]) / (n * n)
	}
	if expected == 1 {
		return 0, false
	}
	return (observed - expected) / (1 - expected), true
}
