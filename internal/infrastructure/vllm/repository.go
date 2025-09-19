package vllm

import (
	"connect-go/internal/domain/vllm"
	"fmt"
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
	key := fmt.Sprintf("%s:%s:%s", namespace, runtimeName, model)
	v, exists := r.store[key]
	if !exists {
		v = vllm.NewVLLM(namespace, runtimeName, model)
		r.store[key] = v
	}
	return v, nil
}

func (r *InMemoryVLLMRepository) Save(vllm *vllm.VLLM) error {
	key := fmt.Sprintf("%s:%s:%s", vllm.Namespace, vllm.RuntimeName, vllm.Model)
	r.store[key] = vllm
	return nil
}
