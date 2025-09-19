package vllm

import (
	domain "connect-go/internal/domain/vllm"
	infra "connect-go/internal/infrastructure/vllm"
)

type VLLMService interface {
	Start(model string) error
	Stop(model string) error
	Update(model string) error
}

type VLLMRepository interface {
	FindByModel(model string) (*domain.VLLM, error)
	Save(vllm *domain.VLLM) error
}

type VLLMServiceImpl struct {
	api  *infra.VLLMAPI
	repo VLLMRepository
}

func (s *VLLMServiceImpl) Start(model string) error {
	vllm, err := s.repo.FindByModel(model)
	if err != nil {
		return err
	}
	if err := vllm.Start(); err != nil {
		return err
	}
	if err := s.api.Start(model); err != nil {
		return err
	}
	return s.repo.Save(vllm)
}

func (s *VLLMServiceImpl) Stop(model string) error {
	vllm, err := s.repo.FindByModel(model)
	if err != nil {
		return err
	}
	if err := vllm.Stop(); err != nil {
		return err
	}
	if err := s.api.Stop(model); err != nil {
		return err
	}
	return s.repo.Save(vllm)
}

func (s *VLLMServiceImpl) Update(model string) error {
	vllm, err := s.repo.FindByModel(model)
	if err != nil {
		return err
	}
	if err := vllm.Update(); err != nil {
		return err
	}
	if err := s.api.Update(model); err != nil {
		return err
	}
	return s.repo.Save(vllm)
}
