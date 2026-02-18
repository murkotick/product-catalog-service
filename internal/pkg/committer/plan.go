package committer

import "cloud.google.com/go/spanner"

type Plan struct {
	mutations []*spanner.Mutation
}

func NewPlan() *Plan {
	return &Plan{
		mutations: make([]*spanner.Mutation, 0),
	}
}

func (p *Plan) Add(m *spanner.Mutation) {
	if m == nil {
		return
	}
	p.mutations = append(p.mutations, m)
}

func (p *Plan) IsEmpty() bool {
	return len(p.mutations) == 0
}

func (p *Plan) Mutations() []*spanner.Mutation {
	return p.mutations
}
