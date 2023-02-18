package controllers

import (
	"context"
	"fmt"
	reloadrestarttriggerv1alpha1 "reload-restart-trigger/api/v1alpha1"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const rolloutAnnotationName = "rollout-operator.k8s.faith/restartedAt"

func updateTriggerStatus(reqLogger logr.Logger, triggerStatus *reloadrestarttriggerv1alpha1.TriggerStatus, r *ResourceReloadRestartTriggerReconciler, namespace string) error {
	reqLogger.V(4).Info("Status before updateTriggerStatus change", "triggerStatus", triggerStatus)
	kind := triggerStatus.Kind
	name := triggerStatus.Name
	resource := &unstructured.Unstructured{}
	resource.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "",
		Kind:    kind,
		Version: "v1",
	})
	err := r.Client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, resource)
	if err == nil {
		triggerStatus.ResourceVersion = resource.GetResourceVersion()
		triggerStatus.State = "Found"
	} else if errors.IsNotFound(err) {
		triggerStatus.State = "NotFound"
	} else {
		reqLogger.Error(err, "Unable to get ResourceVersion")
		triggerStatus.State = err.Error()
		return err
	}
	reqLogger.V(4).Info("Status after updateTriggerStatus change", "triggerStatus", triggerStatus)
	return nil
}

func updateTargetStatus(reqLogger logr.Logger, targetStatus *reloadrestarttriggerv1alpha1.TargetStatus, r *ResourceReloadRestartTriggerReconciler, triggerStatuses []reloadrestarttriggerv1alpha1.TriggerStatus, namespace string) error {
	kind := targetStatus.Kind
	name := targetStatus.Name
	reqLogger = reqLogger.WithValues("function", "updateTargetStatus", "target.namespace", namespace, "target.kind", kind, "target.name", name)
	reqLogger.V(3).Info("Updating target status")

	reqLogger.V(2).Info("Check if any triggers have changed")
	targetTriggerStatuses := targetStatus.Triggers
	reqLogger.V(4).Info("Trigger status before check", "targetStatus", targetStatus, "triggerStatuses", triggerStatuses)
	triggersHaveChanged := false
	for idx, triggerStatus := range triggerStatuses {
		targetTriggerStatus := targetTriggerStatuses[idx]
		if targetTriggerStatus.ResourceVersion == "" {
			targetTriggerStatuses[idx] = triggerStatus
		} else if targetTriggerStatus.ResourceVersion != triggerStatus.ResourceVersion {
			triggersHaveChanged = true
		}
	}
	if !triggersHaveChanged {
		reqLogger.V(2).Info("Nothing to do as no triggers have changed")
		return nil
	}

	// Check if the target can be found i Kubernetes
	target := &unstructured.Unstructured{}
	target.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1",
		Kind:    kind,
	})
	err := r.Client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, target)

	if err != nil {
		reqLogger.Error(err, "Target not found")
		if errors.IsNotFound(err) {
			targetStatus.State = "NotFound"
			targetStatus.Triggers = triggerStatuses
			return nil
		}
		targetStatus.State = "LookupError"
		return err
	}

	reqLogger.V(3).Info("Target found")
	targetStatus.State = "Found"

	reqLogger.Info("There have been changes to triggers and we found the target => restart the target")
	mergePatch := []byte(fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"%s":"%s"}}}}}`, rolloutAnnotationName, time.Now().Format(time.RFC3339)))
	err = r.Client.Patch(context.TODO(), target, client.RawPatch(types.MergePatchType, mergePatch))
	if err == nil {
		targetStatus.Triggers = triggerStatuses
		reqLogger.Info("Restart succeeded")
	} else {
		reqLogger.Error(err, "Restart failed")
	}
	return err
}

func reconcileCrd(reqLogger logr.Logger, r *ResourceReloadRestartTriggerReconciler, request reconcile.Request, crd *reloadrestarttriggerv1alpha1.ResourceReloadRestartTrigger) error {
	namespace := crd.Namespace
	crdName := crd.Name
	triggers := crd.Spec.Triggers
	reqLogger = reqLogger.WithValues("CRD.Namespace", namespace, "CRD.Name", crdName)

	reqLogger.V(4).Info("Start reconcileCrd", "status", crd.Status)

	reqLogger.V(3).Info("The request has to match this CRD or something we watch for us to actually do a job")
	isRelevant := false

	reqName := request.Name
	if reqName == crdName {
		isRelevant = true
		reqLogger.Info("The requests affects the CRD and is relevant for us")
	} else {
		for idx := range triggers {
			if reqName == triggers[idx].Name {
				isRelevant = true
				reqLogger.Info("The request affects a trigger and is relevant for us")
			}
		}
	}
	if !isRelevant {
		reqLogger.V(3).Info("The request is not relevant as the request object isn't related to a trigger or the CRD")
		return nil
	}

	// Ensure that all status fields are present and in the correct order
	triggerStatuses, triggerStatusOrderChanged := ensureTriggerStatusOrder(reqLogger, crd.Status.Triggers, crd.Spec.Triggers)
	targetStatuses, targetStatusOrderChanged := ensureTargetStatusOrder(reqLogger, crd.Status.Targets, crd.Spec.Targets)
	reqLogger.V(3).Info("The result of updating status ordering", "Trigger status order changed", triggerStatusOrderChanged, "Target status order changed", targetStatusOrderChanged)

	for idx := range targetStatuses {
		targetStatus := &targetStatuses[idx]
		targetTriggerStatus, targetTriggerStatusChanged := ensureTriggerStatusOrder(reqLogger, targetStatus.Triggers, crd.Spec.Triggers)
		if targetTriggerStatusChanged {
			reqLogger.V(4).Info("Update status for trigger for a target", "targetstatus", targetStatus, "targetTriggerStatus", targetTriggerStatus)
			targetStatus.Triggers = targetTriggerStatus
		}
	}

	// Update status for each trigger
	var errStatusUpdate error
	for idx := range triggerStatuses {
		err := updateTriggerStatus(reqLogger, &triggerStatuses[idx], r, namespace)
		if err != nil {
			reqLogger.Error(err, "Updating trigger status failed", "triggerstatus", triggerStatuses[idx])
			if errStatusUpdate == nil {
				errStatusUpdate = err
			}
		}
	}
	crd.Status.Triggers = triggerStatuses

	// Update status for each target. Targets will be restarted if needed
	reqLogger.V(4).Info("Got targetstatus", "length", len(targetStatuses), "statuses", targetStatuses)
	errStatusUpdate = nil
	for idx := range targetStatuses {
		err := updateTargetStatus(reqLogger, &targetStatuses[idx], r, triggerStatuses, namespace)
		if err != nil {
			reqLogger.Error(err, "Updating targetstatus failed", "targetStatus", targetStatuses[idx])
			if errStatusUpdate == nil {
				errStatusUpdate = err
			}
		}
	}
	crd.Status.Targets = targetStatuses

	reqLogger.V(3).Info("Updating target statuses", "targetstatuses", targetStatuses)
	crd.Status.Targets = targetStatuses
	errCrd := r.Client.Status().Update(context.TODO(), crd)
	if errCrd != nil {
		reqLogger.Error(errCrd, "Failed to update crd status")
		return errCrd
	}
	return errStatusUpdate
}

func isEqualTriggerStatusOrder(triggerStatuses []reloadrestarttriggerv1alpha1.TriggerStatus, triggers []reloadrestarttriggerv1alpha1.TriggerReference) bool {
	if len(triggerStatuses) != len(triggers) {
		return false
	}
	for idx := range triggerStatuses {
		trigger := triggers[idx]
		if trigger.Kind != triggerStatuses[idx].Kind {
			return false
		}
		if trigger.Name != triggerStatuses[idx].Name {
			return false
		}
	}
	return true
}
func ensureTriggerStatusOrder(reqLogger logr.Logger, triggerStatuses []reloadrestarttriggerv1alpha1.TriggerStatus, triggers []reloadrestarttriggerv1alpha1.TriggerReference) ([]reloadrestarttriggerv1alpha1.TriggerStatus, bool) {
	if isEqualTriggerStatusOrder(triggerStatuses, triggers) {
		return triggerStatuses, false
	}
	m := make(map[string]reloadrestarttriggerv1alpha1.TriggerStatus)
	for _, triggerStatus := range triggerStatuses {
		key := fmt.Sprintf("%s/%s", triggerStatus.Kind, triggerStatus.Name)
		m[key] = triggerStatus
	}
	triggerStatuses = []reloadrestarttriggerv1alpha1.TriggerStatus{}
	for _, trigger := range triggers {
		key := fmt.Sprintf("%s/%s", trigger.Kind, trigger.Name)
		triggerStatus := m[key]
		if triggerStatus.Kind == "" {
			triggerStatus.Kind = trigger.Kind
			triggerStatus.Name = trigger.Name
		}
		triggerStatuses = append(triggerStatuses, triggerStatus)
	}
	return triggerStatuses, true
}

func isEqualTargetStatusOrder(targetStatuses []reloadrestarttriggerv1alpha1.TargetStatus, targets []reloadrestarttriggerv1alpha1.TargetReference) bool {
	if len(targetStatuses) != len(targets) {
		return false
	}
	for idx, targetStatus := range targetStatuses {
		target := targets[idx]
		if targetStatus.Kind != target.Kind {
			return false
		}
		if targetStatus.Name != target.Name {
			return false
		}
	}
	return true
}
func ensureTargetStatusOrder(reqLogger logr.Logger, targetStatuses []reloadrestarttriggerv1alpha1.TargetStatus, targets []reloadrestarttriggerv1alpha1.TargetReference) ([]reloadrestarttriggerv1alpha1.TargetStatus, bool) {
	if isEqualTargetStatusOrder(targetStatuses, targets) {
		return targetStatuses, false
	}
	m := make(map[string]reloadrestarttriggerv1alpha1.TargetStatus)
	for _, targetStatus := range targetStatuses {
		key := fmt.Sprintf("%s/%s", targetStatus.Kind, targetStatus.Name)
		m[key] = targetStatus
	}
	targetStatuses = []reloadrestarttriggerv1alpha1.TargetStatus{}
	for _, target := range targets {
		key := fmt.Sprintf("%s/%s", target.Kind, target.Name)
		targetStatus := m[key]
		if targetStatus.Kind == "" {
			targetStatus.Kind = target.Kind
			targetStatus.Name = target.Name
			targetStatus.State = "NotChecked"
		}
		targetStatuses = append(targetStatuses, targetStatus)
	}
	return targetStatuses, true
}
