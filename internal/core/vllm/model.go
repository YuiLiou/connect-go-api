package vllm

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Status string

const (
	StatusRunning  Status = "Running"
	StatusStopped  Status = "Stopped"
	StatusUpdating Status = "Updating"
	StatusFailed   Status = "Failed"
	StatusPending  Status = "Pending"
)

const (
	ActionStart  = "start"
	ActionStop   = "stop"
	ActionUpdate = "update"
)

type VLLMResource struct {
	Name  string
	Model string
	Phase string
}

type VLLMUseCase struct {
	Model       string
	Status      Status
	Namespace   string
	RuntimeName string
}

// VLLMStatus represents the status of a VLLM CR
type VLLMStatus struct {
	Phase         string
	Message       string
	StartTime     metav1.Time
	Condition     VLLMCondition
	ReadyReplicas int
}

// VLLMCondition represents a status condition
type VLLMCondition struct {
	Type               string
	Status             string
	LastTransitionTime metav1.Time
	Reason             string
	Message            string
}

type VLLMCR struct {
	APIVersion string                 `yaml:"apiVersion"`
	Kind       string                 `yaml:"kind"`
	Metadata   map[string]string      `yaml:"metadata"`
	Spec       map[string]interface{} `yaml:"spec"`
}

func NewVLLM(namespace, runtimeName, model string) *VLLMUseCase {
	return &VLLMUseCase{
		Namespace:   namespace,
		RuntimeName: runtimeName,
		Model:       model,
	}
}

func (v *VLLMUseCase) Start() error {
	if v.Status == StatusRunning {
		return fmt.Errorf("model %s is already running", v.Model)
	}
	v.Status = StatusRunning
	return nil
}

func (v *VLLMUseCase) Stop() error {
	if v.Status == StatusStopped {
		return fmt.Errorf("model %s is already stopped", v.Model)
	}
	v.Status = StatusStopped
	return nil
}

func (v *VLLMUseCase) Update() error {
	if v.Status == StatusUpdating {
		return fmt.Errorf("model %s is already updating", v.Model)
	}
	if v.Status != StatusRunning {
		return fmt.Errorf("model %s must be running to update", v.Model)
	}
	v.Status = StatusUpdating
	return nil
}
