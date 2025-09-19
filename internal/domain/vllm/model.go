package vllm

import "fmt"

type Status string

const (
	StatusRunning  Status = "Running"
	StatusStopped  Status = "Stopped"
	StatusUpdating Status = "Updating"
	StatusFailed   Status = "Failed"
)

type VLLM struct {
	Model  string
	Status Status
}

func NewVLLM(model string) *VLLM {
	return &VLLM{Model: model, Status: StatusStopped}
}

func (v *VLLM) Start() error {
	if v.Status == StatusRunning {
		return fmt.Errorf("model %s is already running", v.Model)
	}
	v.Status = StatusRunning
	return nil
}

func (v *VLLM) Stop() error {
	if v.Status == StatusStopped {
		return fmt.Errorf("model %s is already stopped", v.Model)
	}
	v.Status = StatusStopped
	return nil
}

func (v *VLLM) Update() error {
	if v.Status == StatusUpdating {
		return fmt.Errorf("model %s is already updating", v.Model)
	}
	if v.Status != StatusRunning {
		return fmt.Errorf("model %s must be running to update", v.Model)
	}
	v.Status = StatusUpdating
	return nil
}
