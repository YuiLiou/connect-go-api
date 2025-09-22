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
	if err := s.api.Start(model); err != nil {
		return nil, err
	}
	refreshVLLM, err := s.repo.FindByModel(namespace, runningName, model)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh VLLM status after start: %w", err)
	}
	return refreshVLLM, nil
}

func (s *VLLMServiceImpl) Stop(namespace, runningName, model string) (*domain.VLLM, error) {
	vllm, err := s.repo.FindByModel(namespace, runningName, model)
	if err != nil {
		return nil, err
	}
	if vllm.Status == domain.StatusStopped {
		return nil, fmt.Errorf("model %s is already stopped", model)
	}
	if err := s.api.Stop(model); err != nil {
		return nil, err
	}
	refreshVLLM, err := s.repo.FindByModel(namespace, runningName, model)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh VLLM status after stop: %w", err)
	}
	return refreshVLLM, nil
}

func (s *VLLMServiceImpl) Update(namespace, runningName, model string) (*domain.VLLM, error) {
	_, err := s.repo.FindByModel(namespace, runningName, model)
	if err != nil {
		return nil, err
	}
	if err := s.api.Update(model); err != nil {
		return nil, err
	}
	refreshVLLM, err := s.repo.FindByModel(namespace, runningName, model)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh VLLM status after update: %w", err)
	}
	return refreshVLLM, nil
}
