package cases

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/smpebble/hf-readmit-agent/internal/domain"
)

type Repository struct{ cases map[string]domain.Case }

func Load(path string) (*Repository, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read dataset: %w", err)
	}
	var dataset domain.Dataset
	if err := json.Unmarshal(contents, &dataset); err != nil {
		return nil, fmt.Errorf("decode dataset: %w", err)
	}
	repo := &Repository{cases: make(map[string]domain.Case, len(dataset.Cases))}
	for _, item := range dataset.Cases {
		repo.cases[item.CaseID] = item
	}
	return repo, nil
}

func (r *Repository) List() []domain.Case {
	items := make([]domain.Case, 0, len(r.cases))
	for _, item := range r.cases {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CaseID < items[j].CaseID })
	return items
}

func (r *Repository) Get(caseID string) (domain.Case, bool) {
	item, ok := r.cases[caseID]
	return item, ok
}
