package controllers

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	app "connect-go/internal/application/vllm"
)

type VLLMReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	VLLMService app.VLLMService // 注入 application 層 service
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

	// 2. 根據 action 執行 vLLM production stack 操作
	var err error
	switch cr.Spec.Action {
	case "start":
		_, err = r.VLLMService.Start(cr.Spec.Model, cr.Spec.Action, "")
		if err == nil {
			cr.Status.Phase = "Running"
			cr.Status.Message = "vLLM model started successfully"
		} else {
			cr.Status.Phase = "Failed"
			cr.Status.Message = fmt.Sprintf("Failed to start model: %v", err)
		}
	case "stop":
		_, err = r.VLLMService.Stop(cr.Spec.Model, cr.Spec.Action, "")
		if err == nil {
			cr.Status.Phase = "Stopped"
			cr.Status.Message = "vLLM model stopped successfully"
		} else {
			cr.Status.Phase = "Failed"
			cr.Status.Message = fmt.Sprintf("Failed to stop model: %v", err)
		}
	case "update":
		_, err = r.VLLMService.Update(cr.Spec.Model, cr.Spec.Action, "")
		if err == nil {
			cr.Status.Phase = "Updated"
			cr.Status.Message = "vLLM model updated successfully"
		} else {
			cr.Status.Phase = "Failed"
			cr.Status.Message = fmt.Sprintf("Failed to update model: %v", err)
		}
	default:
		cr.Status.Phase = "Unknown"
		cr.Status.Message = fmt.Sprintf("Unknown action: %s", cr.Spec.Action)
	}

	// 3. Update CR status
	if updateErr := r.Client.Status().Update(ctx, &cr); updateErr != nil {
		log.Error(updateErr, "failed to update CR status")
		return ctrl.Result{}, updateErr
	}

	log.Info("Reconciled VLLMCR", "action", cr.Spec.Action, "phase", cr.Status.Phase, "message", cr.Status.Message)
	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager
func (r *VLLMReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&VLLMCR{}).
		Complete(r)
}
