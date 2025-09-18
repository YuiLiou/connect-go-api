package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type VLLMReconciler struct {
	client.Client
	Schema *runtime.Scheme
}

type VLLMCR struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec struct {
		Model  string `json:"model"`
		Action string `json:"action"` // start, stop, update
	} `json:"spec"`
	Status struct {
		Phase   string `json:"phase"`
		Message string `json:"message"`
	} `json:"status"`
}

// DeepCopyObject is required to implement client.Object
func (in *VLLMCR) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := new(VLLMCR)
	*out = *in
	out.ObjectMeta = *in.ObjectMeta.DeepCopy()
	return out
}

func (r *VLLMReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	// 1. Fetch the VLLMCR
	var cr VLLMCR
	if err := r.Get(ctx, req.NamespacedName, &cr); err != nil {
		log.Error(err, "unable to fetch VLLMCR", "namespace", req.Namespace, "name", req.Name)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	// 2. Call the Go server API (vLLM service)
	payload, err := json.Marshal(map[string]string{
		"model":  cr.Spec.Model,
		"action": cr.Spec.Action,
	})
	if err != nil {
		log.Error(err, "failed to marshal API payload")
		cr.Status.Phase = "Failed"
		cr.Status.Message = fmt.Sprintf("Payload error: %v", err)
		if updateErr := r.Client.Status().Update(ctx, &cr); updateErr != nil {
			log.Error(updateErr, "failed to update CR status")
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, err
	}
	resp, err := http.Post("http://go-server-service:8799/vllm/update", "application/json", bytes.NewBuffer(payload))
	if err != nil {
		log.Error(err, "failed to call vLLM API")
		cr.Status.Phase = "Failed"
		cr.Status.Message = fmt.Sprintf("API error: %v", err)
		if updateErr := r.Client.Status().Update(ctx, &cr); updateErr != nil {
			log.Error(updateErr, "failed to update CR status")
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, err
	}
	defer resp.Body.Close()
	// 3. Update CR status based on API response
	if resp.StatusCode == http.StatusOK {
		cr.Status.Phase = "Succeeded"
		cr.Status.Message = "vLLM updated successfully"
	} else {
		cr.Status.Phase = "Failed"
		cr.Status.Message = fmt.Sprintf("API returned status: %s", resp.Status)
	}
	// 4. Update the CR status
	if err := r.Client.Status().Update(ctx, &cr); err != nil {
		log.Error(err, "failed to update CR status")
		return ctrl.Result{}, err
	}
	log.Info("Reconciled VLLMCR", "phase", cr.Status.Phase, "message", cr.Status.Message)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *VLLMReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&VLLMCR{}).
		Complete(r)
}
