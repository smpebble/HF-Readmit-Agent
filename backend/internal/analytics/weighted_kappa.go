package analytics

var TierOrder = []string{"L0", "L1", "L2", "L3"}

// LinearWeightedKappa accounts for the ordinal distance between tiers.
func LinearWeightedKappa(left, right []string) (float64, bool) {
	if len(left) == 0 || len(left) != len(right) {
		return 0, false
	}
	index := tierIndex()
	observed := make([][]int, len(TierOrder))
	leftCounts, rightCounts := make([]int, len(TierOrder)), make([]int, len(TierOrder))
	for i := range observed {
		observed[i] = make([]int, len(TierOrder))
	}
	for i := range left {
		l, lok := index[left[i]]
		r, rok := index[right[i]]
		if !lok || !rok {
			return 0, false
		}
		observed[l][r]++
		leftCounts[l]++
		rightCounts[r]++
	}
	n := float64(len(left))
	observedScore, expectedScore := 0.0, 0.0
	for i := range TierOrder {
		for j := range TierOrder {
			weight := 1 - float64(abs(i-j))/float64(len(TierOrder)-1)
			observedScore += weight * float64(observed[i][j]) / n
			expectedScore += weight * float64(leftCounts[i]*rightCounts[j]) / (n * n)
		}
	}
	if expectedScore == 1 {
		return 0, false
	}
	return (observedScore - expectedScore) / (1 - expectedScore), true
}

func ConfusionMatrix(left, right []string) ([][]int, bool) {
	if len(left) != len(right) {
		return nil, false
	}
	index := tierIndex()
	matrix := make([][]int, len(TierOrder))
	for i := range matrix {
		matrix[i] = make([]int, len(TierOrder))
	}
	for i := range left {
		l, lok := index[left[i]]
		r, rok := index[right[i]]
		if !lok || !rok {
			return nil, false
		}
		matrix[l][r]++
	}
	return matrix, true
}
func tierIndex() map[string]int { return map[string]int{"L0": 0, "L1": 1, "L2": 2, "L3": 3} }
func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
