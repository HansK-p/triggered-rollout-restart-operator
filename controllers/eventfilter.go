package controllers

import (
	"context"
	"fmt"
	reloadrestarttriggerv1alpha1 "reload-restart-trigger/api/v1alpha1"
	"strconv"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type eventObject struct {
	metav1.TypeMeta
	metav1.ObjectMeta
}

func findCRsMatchingEvent(logger logr.Logger, cli client.Client, eventObject client.Object) ([]*reloadrestarttriggerv1alpha1.ResourceReloadRestartTrigger, error) {
	crs := []*reloadrestarttriggerv1alpha1.ResourceReloadRestartTrigger{}
	logger = logger.WithValues("CRKind", "reloadrestarttriggerv1alpha1.ResourceReloadRestartTrigger")

	// Check if there is a at least one ResourceReloadRestartTrigger in the same namespace
	instances := reloadrestarttriggerv1alpha1.ResourceReloadRestartTriggerList{}
	err := cli.List(context.TODO(), &instances, client.InNamespace(eventObject.GetNamespace()))
	if err != nil {
		logger.V(0).Error(err, "Error returned when listing CRs in the namespace")
		return crs, err
	}

	if len(instances.Items) == 0 {
		logger.V(2).Info("There are no CRs in the same namespace as the event object so this event is not of interest")
		return crs, nil
	}

	eventObjectKind := ""
	if _, ok := eventObject.(*corev1.Secret); ok {
		eventObjectKind = "Secret"
	} else if _, ok := eventObject.(*corev1.ConfigMap); ok {
		eventObjectKind = "ConfigMap"
	} else {
		err := fmt.Errorf("The event object is not a Secret and not a ConfigMap")
		logger.V(0).Error(err, "The event object is not a secret and not a ConfigMap")
		return crs, nil
	}
	logger = logger.WithValues("EventObjectKind", eventObjectKind)
	// Check if any of the returned CRs are relevant for the event object
	for idx := range instances.Items {
		instance := &instances.Items[idx]
		logger := logger.WithValues("CRName", instance.Name)
		logger.V(2).Info("Checking the instance")
		for _, trigger := range instance.Spec.Triggers {
			logger := logger.WithValues("Trigger.Kind", trigger.Kind)
			if trigger.Kind == eventObjectKind && trigger.Name == eventObject.GetName() {
				logger.V(1).Info("This CR has the event object as a trigger")
				crs = append(crs, instance)
			} else {
				logger.V(1).Info("This CR doesn not have the event object as a trigger")
			}
		}
	}
	return crs, nil
}

func filterEvents(logger logr.Logger, cli client.Client, eventObj client.Object) bool {
	logger.V(3).Info("Checking event object")
	crs, err := findCRsMatchingEvent(logger, cli, eventObj)
	if err != nil {
		logger.V(0).Error(err, "There was an error finding CRs with event object as a trigger")
		return false
	}
	if len(crs) > 0 {
		logger.V(1).Info("The event object matched at least one source rule. Handle it.", "matches", len(crs))
		return true
	}
	logger.V(3).Info("The event object did not match any source CR. Skip it.")
	return false
}

// Only events matching ResourceReloadRestartCR will result in a reconciler event
func eventFilter(logger logr.Logger, cli client.Client) predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			logger := logger.WithValues("Event", "Create", "Namespace", e.Object.GetNamespace(), "EventObjectName", e.Object.GetName())
			return filterEvents(logger, cli, e.Object)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			logger := logger.WithValues("Event", "Delete", "Namespace", e.Object.GetNamespace(), "EventObjectName", e.Object.GetName())
			return filterEvents(logger, cli, e.Object)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			logger := logger.WithValues("Event", "Update", "Namespace", e.ObjectNew.GetNamespace(), "EventObjectName", e.ObjectNew.GetName())
			return filterEvents(logger, cli, e.ObjectNew)
		},
		GenericFunc: func(e event.GenericEvent) bool {
			logger := logger.WithValues("Event", "Generic", "Namespace", e.Object.GetNamespace(), "EventObjectName", e.Object.GetName())
			return filterEvents(logger, cli, e.Object)
		},
	}
}

func eventHandler(logger logr.Logger, cli client.Client) handler.MapFunc {
	return func(a client.Object) []reconcile.Request {
		logger := logger.WithValues(
			"function", "configMapEventHandler",
			"namespace", a.GetNamespace(),
			"name", a.GetName(),
		)
		logger.V(3).Info("Do a recounciler fan-out for the event")
		requests := []reconcile.Request{}
		crs, err := findCRsMatchingEvent(logger, cli, a)
		if err != nil {
			logger.V(0).Error(err, "Error retrieving CRs")
			return requests
		}

		for idx, cr := range crs {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      cr.Name,
					Namespace: cr.Namespace,
				},
			})
			logger.V(1).Info("Create recouncile request",
				"name", cr.Name,
				"namespace", cr.Namespace,
				"kind", cr.Kind,
				"idx", strconv.Itoa(idx),
			)
		}
		return requests
	}
}
