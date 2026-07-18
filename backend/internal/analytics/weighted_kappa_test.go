package analytics

import "testing"

func TestLinearWeightedKappaAndConfusionMatrix(t *testing.T) {
	left := []string{"L0", "L1", "L2", "L3"}
	right := []string{"L0", "L2", "L2", "L3"}
	kappa, defined := LinearWeightedKappa(left, right)
	if !defined || kappa <= 0 {
		t.Fatalf("expected positive weighted kappa, got %v", kappa)
	}
	matrix, defined := ConfusionMatrix(left, right)
	if !defined || matrix[0][0] != 1 || matrix[1][2] != 1 || matrix[2][2] != 1 || matrix[3][3] != 1 {
		t.Fatalf("unexpected matrix: %#v", matrix)
	}
}
