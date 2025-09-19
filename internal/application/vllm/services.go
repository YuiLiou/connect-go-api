package vllm

import (
	"connect-go/internal/domain/vllm"
	domain "connect-go/internal/domain/vllm"
	infra "connect-go/internal/infrastructure/vllm"
)

type VLLMService interface {
	Start(model string) error
	Stop(model string) error
	Update(model string) error
	GetStatus(model string) (domain.Status, error)
}

type VLLMRepository interface {
	FindByModel(model string) (*domain.VLLM, error)
	Save(vllm *domain.VLLM) error
}

type VLLMServiceImpl struct {
	api  *infra.VLLMAPI
	repo VLLMRepository
}

func NewVLLMServiceImpl(api *infra.VLLMAPI, repo VLLMRepository) *VLLMServiceImpl {
	return &VLLMServiceImpl{api: api, repo: repo}
}

func (s *VLLMServiceImpl) executeOperation(model string, domainOp func(*vllm.VLLM) error, apiOp func(string) error) error {
	vllm, err := s.repo.FindByModel(model)
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

func (s *VLLMServiceImpl) Start(model string) error {
	return s.executeOperation(model, (*vllm.VLLM).Start, s.api.Start)
}

func (s *VLLMServiceImpl) Stop(model string) error {
	return s.executeOperation(model, (*vllm.VLLM).Stop, s.api.Stop)
}

func (s *VLLMServiceImpl) Update(model string) error {
	return s.executeOperation(model, (*vllm.VLLM).Update, s.api.Update)
}

func (s *VLLMServiceImpl) GetStatus(model string) (domain.Status, error) {
	v, err := s.repo.FindByModel(model)
	if err != nil {
		return domain.StatusFailed, err
	}
	return v.Status, nil
}
