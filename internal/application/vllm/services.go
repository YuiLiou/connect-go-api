package vllm

import (
	"connect-go/internal/domain/vllm"
	infra "connect-go/internal/infrastructure/vllm"
)

type VLLMService interface {
	Start(namespace, runtimeName, model string) (*vllm.VLLM, error)
	Stop(namespace, runtimeName, model string) (*vllm.VLLM, error)
	Update(namespace, runtimeName, model string) (*vllm.VLLM, error)
}

type VLLMServiceImpl struct {
	api  *infra.VLLMAPI
	repo infra.VLLMRepository
}

func NewVLLMServiceImpl(api *infra.VLLMAPI, repo infra.VLLMRepository) *VLLMServiceImpl {
	return &VLLMServiceImpl{api: api, repo: repo}
}

func (s *VLLMServiceImpl) executeOperation(
	namespace, runtimeName, model string,
	domainOp func(*vllm.VLLM) error,
	apiOp func(string) error,
) (*vllm.VLLM, error) {
	vllm, err := s.repo.FindByModel(namespace, runtimeName, model)
	if err != nil {
		return nil, err
	}
	if err := domainOp(vllm); err != nil {
		return nil, err
	}
	if err := apiOp(model); err != nil {
		return nil, err
	}
	if err := s.repo.Save(vllm); err != nil {
		return nil, err
	}
	return vllm, nil
}

func (s *VLLMServiceImpl) Start(namespace, runtimeName, model string) (*vllm.VLLM, error) {
	return s.executeOperation(namespace, runtimeName, model, (*vllm.VLLM).Start, s.api.Start)
}

func (s *VLLMServiceImpl) Stop(namespace, runtimeName, model string) (*vllm.VLLM, error) {
	return s.executeOperation(namespace, runtimeName, model, (*vllm.VLLM).Stop, s.api.Stop)
}

func (s *VLLMServiceImpl) Update(namespace, runtimeName, model string) (*vllm.VLLM, error) {
	return s.executeOperation(namespace, runtimeName, model, (*vllm.VLLM).Update, s.api.Update)
}
