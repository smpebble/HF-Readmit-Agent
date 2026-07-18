package analytics

import "testing"

func TestCohenKappa(t *testing.T) {
	kappa, defined := CohenKappa([]string{"L0", "L1", "L0", "L1"}, []string{"L0", "L1", "L1", "L0"})
	if !defined {
		t.Fatal("expected defined kappa")
	}
	if kappa != 0 {
		t.Fatalf("kappa = %v, want 0", kappa)
	}
}

func TestCohenKappaRejectsUndefinedInput(t *testing.T) {
	if _, defined := CohenKappa([]string{"L0"}, []string{"L0"}); defined {
		t.Fatal("single-category perfect agreement is undefined")
	}
}
