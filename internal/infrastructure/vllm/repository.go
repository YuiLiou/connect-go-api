package vllm

import (
	"connect-go/internal/domain/vllm"
)

type VLLMRepository interface {
	FindByModel(namespace, runtimeName, model string) (*vllm.VLLM, error)
	Save(vllm *vllm.VLLM) error
}

type InMemoryVLLMRepository struct {
	store map[string]*vllm.VLLM
}

func NewInMemoryVLLMRepository() *InMemoryVLLMRepository {
	return &InMemoryVLLMRepository{
		store: make(map[string]*vllm.VLLM),
	}
}

func (r *InMemoryVLLMRepository) FindByModel(namespace, runtimeName, model string) (*vllm.VLLM, error) {
	v, exists := r.store[model]
	if !exists {
		v = vllm.NewVLLM(namespace, runtimeName, model)
		r.store[model] = v
	}
	return v, nil
}

func (r *InMemoryVLLMRepository) Save(vllm *vllm.VLLM) error {
	r.store[vllm.Model] = vllm
	return nil
}
