package vllm

import (
	"connect-go/internal/domain/vllm"
	domain "connect-go/internal/domain/vllm"
	infra "connect-go/internal/infrastructure/vllm"
)

type VLLMService interface {
	Start(namespace, runtimeName, model string) error
	Stop(namespace, runtimeName, model string) error
	Update(namespace, runtimeName, model string) error
	GetStatus(namespace, runtimeName, model string) (domain.Status, error)
}

type VLLMServiceImpl struct {
	api  *infra.VLLMAPI
	repo infra.VLLMRepository
}

func NewVLLMServiceImpl(api *infra.VLLMAPI, repo infra.VLLMRepository) *VLLMServiceImpl {
	return &VLLMServiceImpl{api: api, repo: repo}
}

func (s *VLLMServiceImpl) executeOperation(namespace, runtimeName, model string, domainOp func(*vllm.VLLM) error, apiOp func(string) error) error {
	vllm, err := s.repo.FindByModel(namespace, runtimeName, model)
	if err != nil {
		return err
	}
	if err := domainOp(vllm); err != nil {
		return err
	}
	if err := apiOp(model); err != nil {
		return err
	}
	return s.repo.Save(vllm)
}

func (s *VLLMServiceImpl) Start(namespace, runtimeName, model string) error {
	return s.executeOperation(namespace, runtimeName, model, (*vllm.VLLM).Start, s.api.Start)
}

func (s *VLLMServiceImpl) Stop(namespace, runtimeName, model string) error {
	return s.executeOperation(namespace, runtimeName, model, (*vllm.VLLM).Stop, s.api.Stop)
}

func (s *VLLMServiceImpl) Update(namespace, runtimeName, model string) error {
	return s.executeOperation(namespace, runtimeName, model, (*vllm.VLLM).Update, s.api.Update)
}

func (s *VLLMServiceImpl) GetStatus(namespace, runtimeName, model string) (domain.Status, error) {
	v, err := s.repo.FindByModel(namespace, runtimeName, model)
	if err != nil {
		return domain.StatusFailed, err
	}
	return v.Status, nil
}
