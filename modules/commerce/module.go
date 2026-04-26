package commerce

import (
	"github.com/anzhiyu-c/anheyu-app/ent"
	"github.com/anzhiyu-c/anheyu-app/pkg/integration/dp7575"
)

type Module struct {
	service *Service
}

func NewModule(client *ent.Client, memberClient *dp7575.Client) *Module {
	repo := NewBindingRepository(client)
	service := NewService(repo, memberClient)
	return &Module{service: service}
}

func (m *Module) Service() *Service {
	return m.service
}
