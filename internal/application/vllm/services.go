package vllm

import (
	domain "connect-go/internal/domain/vllm"
	infra "connect-go/internal/infrastructure/vllm"
	"fmt"
)

type VLLMService interface {
	Start(namespace, runtimeName, model string) (*domain.VLLM, error)
	Stop(namespace, runtimeName, model string) (*domain.VLLM, error)
	Update(namespace, runtimeName, model string) (*domain.VLLM, error)
}

type VLLMServiceImpl struct {
	api  *infra.VLLMAPI
	repo infra.VLLMRepository
}

func NewVLLMServiceImpl(api *infra.VLLMAPI, repo infra.VLLMRepository) *VLLMServiceImpl {
	return &VLLMServiceImpl{
		api:  api,
		repo: repo,
	}
}

func (s *VLLMServiceImpl) Start(namespace, runningName, model string) (*domain.VLLM, error) {
	vllm, err := s.repo.FindByModel(namespace, runningName, model)
	if err != nil {
		return nil, err
	}
	if vllm.Status == domain.StatusRunning {
		return nil, fmt.Errorf("model %s is already running", model)
	}
	if err := s.repo.UpdateCRStatusToStart(namespace, runningName, model); err != nil {
		return nil, err
	}
	vllm.Status = domain.StatusRunning
	return vllm, s.api.Start(model)
}

func (s *VLLMServiceImpl) Stop(namespace, runningName, model string) (*domain.VLLM, error) {
	vllm, err := s.repo.FindByModel(namespace, runningName, model)
	if err != nil {
		return nil, err
	}
	if vllm.Status == domain.StatusStopped {
		return nil, fmt.Errorf("model %s is already stopped", model)
	}
	if err := s.repo.Save(vllm); err != nil {
		return nil, err
	}
	vllm.Status = domain.StatusStopped
	return vllm, s.api.Stop(model)
}

func (s *VLLMServiceImpl) Update(namespace, runningName, model string) (*domain.VLLM, error) {
	vllm, err := s.repo.FindByModel(namespace, runningName, model)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Save(vllm); err != nil {
		return nil, err
	}
	vllm.Status = domain.StatusUpdating
	return vllm, s.api.Update(model)
}
