package resourcereloadrestarttrigger

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
	reloadrestarttriggerv1alpha1 "triggered-rollout-restart-operator/pkg/apis/reloadrestarttrigger/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

// TestResourceReloadRestartTriggerController runs ReconcileResourceReloadRestartTrigger.Reconcile() against a
// fake client
func TestResourceReloadRestartTriggerController(t *testing.T) {
	// Set the logger to development mode for verbose logs.
	logf.SetLogger(logf.ZapLogger(true))

	var (
		name      = "mesourcemeloadmestarttrigger-operator"
		namespace = "mesourcemeloadmestarttrigger"
		triggers  = []reloadrestarttriggerv1alpha1.TriggerReference{
			{Kind: "ConfigMap", Name: "configmap-not-to-be-found"},
			{Kind: "ConfigMap", Name: "configmap-1"},
			{Kind: "Secret", Name: "secret-1"},
			{Kind: "Secret", Name: "secret-not-to-be-found"},
		}
		targets = []reloadrestarttriggerv1alpha1.TargetReference{
			{Kind: "Deployment", Name: "deployment-1"},
			{Kind: "Deployment", Name: "deployment-not-to-be-found"},
			{Kind: "DaemonSet", Name: "daemonset-1"},
			{Kind: "StatefulSet", Name: "statefulset-1"},
		}
	)

	// A ResourceReloadRestartTrigger resource with metadata and spec.
	resourcereloadrestarttrigger := &reloadrestarttriggerv1alpha1.ResourceReloadRestartTrigger{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: reloadrestarttriggerv1alpha1.ResourceReloadRestartTriggerSpec{
			Triggers: triggers,
			Targets:  targets,
		},
	}

	// A Deployment to gracefully restart using annotations
	deployment1 := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-1",
			Namespace: namespace,
		},
	}

	// A DaemonSet to gracefully restart using annotations
	daemonset1 := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "daemonset-1",
			Namespace: namespace,
		},
	}

	// A Statefulset to gracefully restart using annotations
	statefulset1 := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "statefulset-1",
			Namespace: namespace,
		},
	}

	// A configmaps to watch
	configMap1 := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "configmap-1",
			Namespace:       namespace,
			ResourceVersion: "142",
		},
		Data: map[string]string{
			"file1": "content1",
			"file2": "content2",
		},
	}
	configMap2 := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "configmap-2",
			Namespace:       namespace,
			ResourceVersion: "1142",
		},
		Data: map[string]string{
			"file11": "content11",
			"file21": "content21",
		},
	}

	// A secret to watch
	secret1 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "secret-1",
			Namespace:       namespace,
			ResourceVersion: "42",
		},
		Data: map[string][]byte{
			"secret1": []byte("content1"),
			"secret2": []byte("content2"),
		},
	}

	// Objects to track in the fake client.
	objs := []runtime.Object{
		resourcereloadrestarttrigger,
		deployment1,
		daemonset1,
		statefulset1,
		configMap1,
		configMap2,
		secret1,
	}

	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	s.AddKnownTypes(reloadrestarttriggerv1alpha1.SchemeGroupVersion, resourcereloadrestarttrigger)
	// Create a fake client to mock API calls.
	cl := fake.NewFakeClient(objs...)
	// Create a ReconcileResourceReloadRestartTrigger object with the scheme and fake client.
	r := &ReconcileResourceReloadRestartTrigger{client: cl, scheme: s}

	runAndValidateReconcile(t, namespace, name, triggers, targets, r, false, map[string]string{})

	// Update trigger resourceversion
	for _, trigger := range triggers {
		updateResource(t, namespace, trigger.Kind, trigger.Name, r, "before-reconcile-test", "2nd")
	}

	runAndValidateReconcile(t, namespace, name, triggers, targets, r, true, map[string]string{
		"Deployment/deployment-1":               "Present",
		"Deployment/deployment-not-to-be-found": "NotFound",
		"DaemonSet/daemonset-1":                 "Present",
		"StatefulSet/statefulset-1":             "Present",
	})
	removeRolloutAnnotation(t, namespace, targets, r)

	runAndValidateReconcile(t, namespace, name, triggers, targets, r, false, map[string]string{
		"Deployment/deployment-1":               "Present",
		"Deployment/deployment-not-to-be-found": "NotFound",
		"DaemonSet/daemonset-1":                 "Present",
		"StatefulSet/statefulset-1":             "Present",
	})

	// Add a configmap to watch to the crd. This should not trigger a rolling upgrade
	triggers = append(triggers, reloadrestarttriggerv1alpha1.TriggerReference{Kind: "ConfigMap", Name: "configmap-2"})
	updateCRD(t, namespace, name, triggers, targets, r)

	// Run reconcile. This should not trigger a rollout
	runAndValidateReconcile(t, namespace, name, triggers, targets, r, false, map[string]string{
		"Deployment/deployment-1":               "Present",
		"Deployment/deployment-not-to-be-found": "NotFound",
		"DaemonSet/daemonset-1":                 "Present",
		"StatefulSet/statefulset-1":             "Present",
	})

	// Make a change to the latest added configmap
	updateResource(t, namespace, "ConfigMap", "configmap-2", r, "before-reconcile-test", "1st")
	// Run reconcile. This should trigger a rollout
	runAndValidateReconcile(t, namespace, name, triggers, targets, r, true, map[string]string{
		"Deployment/deployment-1":               "Present",
		"Deployment/deployment-not-to-be-found": "NotFound",
		"DaemonSet/daemonset-1":                 "Present",
		"StatefulSet/statefulset-1":             "Present",
	})
	removeRolloutAnnotation(t, namespace, targets, r)

	// Delete triggers that can't be found from the CRD. This should not trigger a rollout
	newTriggers := []reloadrestarttriggerv1alpha1.TriggerReference{}
	for _, trigger := range triggers {
		if strings.Contains(trigger.Name, "not-to-be-found") == false {
			newTriggers = append(newTriggers, trigger)
		}
	}
	if len(triggers) == len(newTriggers) {
		t.Fatalf("Unable to reduce number of triggers when removing triggers not in use, %d != %d", len(triggers), len(newTriggers))
	}
	triggers = newTriggers
	updateCRD(t, namespace, name, triggers, targets, r)
	runAndValidateReconcile(t, namespace, name, triggers, targets, r, false, map[string]string{
		"Deployment/deployment-1":               "Present",
		"Deployment/deployment-not-to-be-found": "NotFound",
		"DaemonSet/daemonset-1":                 "Present",
		"StatefulSet/statefulset-1":             "Present",
	})

	// Remove targets which doesn't exist and reconcile. This should not trigger a rollout
	newTargets := []reloadrestarttriggerv1alpha1.TargetReference{}
	for _, target := range targets {
		if strings.Contains(target.Name, "not-to-be-found") == false {
			newTargets = append(newTargets, target)
		}
	}
	if len(targets) == len(newTargets) {
		t.Fatalf("Unable to reduce number of targets when removing targets that can't be found, %d != %d", len(targets), len(newTargets))
	}
	targets = newTargets
	updateCRD(t, namespace, name, triggers, targets, r)
	runAndValidateReconcile(t, namespace, name, triggers, targets, r, false, map[string]string{
		"Deployment/deployment-1":   "Present",
		"DaemonSet/daemonset-1":     "Present",
		"StatefulSet/statefulset-1": "Present",
	})

	// Change the secret and validate that a rollout is again initiated
	updateResource(t, namespace, "Secret", "secret-1", r, "before-reconcile-test", "3dorsomething")
	runAndValidateReconcile(t, namespace, name, triggers, targets, r, true, map[string]string{
		"Deployment/deployment-1":   "Present",
		"DaemonSet/daemonset-1":     "Present",
		"StatefulSet/statefulset-1": "Present",
	})
}

func runAndValidateReconcile(t *testing.T, namespace, name string,
	triggers []reloadrestarttriggerv1alpha1.TriggerReference,
	targets []reloadrestarttriggerv1alpha1.TargetReference,
	r *ReconcileResourceReloadRestartTrigger,
	expectAnnotation bool,
	expectedTargetStatusStates map[string]string,
) {
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		},
	}
	res, err := r.Reconcile(req)
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}
	// Check the result of reconciliation to make sure it has the desired state.
	if res.Requeue {
		t.Error("reconcile requeue which is not expected after the reconcile")
	}
	validateStatusAfterReconcile(t, namespace, name, triggers, targets, r, expectAnnotation, expectedTargetStatusStates)
}

func updateCRD(t *testing.T, namespace, name string,
	triggers []reloadrestarttriggerv1alpha1.TriggerReference,
	targets []reloadrestarttriggerv1alpha1.TargetReference,
	r *ReconcileResourceReloadRestartTrigger) {
	crd := &reloadrestarttriggerv1alpha1.ResourceReloadRestartTrigger{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, crd)
	if err != nil {
		t.Fatalf("Unable to get CRD, error: %#v", err)
	}
	crd.Spec.Triggers = triggers
	crd.Spec.Targets = targets
	err = r.client.Update(context.TODO(), crd)
	if err != nil {
		t.Fatalf("Unable to update the CRD, the error was: %#v", err)
	}
}

func updateResource(t *testing.T, namespace, kind, name string, r *ReconcileResourceReloadRestartTrigger, annotationName, annotationValue string) {
	resource := &unstructured.Unstructured{}
	resource.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "",
		Kind:    kind,
		Version: "v1",
	})
	err := r.client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, resource)
	if err == nil {
		resource.SetAnnotations(map[string]string{annotationName: annotationValue})
		err = r.client.Update(context.TODO(), resource)
		if err != nil {
			t.Fatalf("Unable to update resources with annotation %s=%s. Error was %#v", annotationName, annotationValue, err)
		}
	} else if errors.IsNotFound(err) == false {
		t.Fatalf("Error retrieving %s with name %s for annotations %s=%s. Got error %#v", kind, name, annotationName, annotationValue, err)
	}
}

func validateStatusAfterReconcile(t *testing.T, namespace, name string,
	triggers []reloadrestarttriggerv1alpha1.TriggerReference,
	targets []reloadrestarttriggerv1alpha1.TargetReference,
	r *ReconcileResourceReloadRestartTrigger,
	expectAnnotation bool,
	expectedTargetStatusStates map[string]string,
) {
	crd := &reloadrestarttriggerv1alpha1.ResourceReloadRestartTrigger{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, crd)
	if err != nil {
		t.Fatalf("Unable to get CRD, error: %#v", err)
	}

	// Create a map of resource versions for our triggers
	triggerResourceVersionMap := make(map[string]string)
	for _, trigger := range triggers {
		kind := trigger.Kind
		name := trigger.Name
		resource := &unstructured.Unstructured{}
		resource.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "",
			Kind:    kind,
			Version: "v1",
		})
		err := r.client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, resource)
		if err == nil {
			triggerResourceVersionMap[fmt.Sprintf("%s/%s", kind, name)] = resource.GetResourceVersion()
		} else if errors.IsNotFound(err) {
			triggerResourceVersionMap[fmt.Sprintf("%s/%s", kind, name)] = ""
		} else {
			t.Fatalf("Got error when getting %s with name %s, error was %#v", kind, name, err)
		}
	}
	// Validate that crd status fields have been updated correctly
	triggerStatuses := crd.Status.Triggers
	if len(triggerStatuses) != len(triggers) {
		t.Fatalf("There is a mismatch between CRD trigger statuses count and triggers count %d != %d", len(triggerStatuses), len(triggers))
	}
	for idx, triggerStatus := range triggerStatuses {
		trigger := &triggers[idx]
		if triggerStatus.Kind != trigger.Kind {
			t.Fatalf("Triggerstatus with idx %d should have been of kind %s, but was of kind %s", idx, trigger.Kind, triggerStatus.Kind)
		}
		if triggerStatus.Name != trigger.Name {
			t.Fatalf("Triggerstatus with idx %d should have had name %s, but had name %s", idx, trigger.Name, triggerStatus.Name)
		}
		expectedResourceVersion := triggerResourceVersionMap[fmt.Sprintf("%s/%s", triggerStatus.Kind, triggerStatus.Name)]
		if expectedResourceVersion != triggerStatus.ResourceVersion {
			t.Fatalf("Trigger status ResourceVersion for %s with name %s should have been '%s', but was '%s'",
				triggerStatus.Kind, triggerStatus.Name, expectedResourceVersion, triggerStatus.ResourceVersion)
		}
		if expectedResourceVersion == "" {
			if triggerStatus.State != "NotFound" {
				t.Fatalf("Trigger status ResourceVersion for %s with name %s should have been '%s', but was '%s'",
					triggerStatus.Kind, triggerStatus.Name, "NotFound", triggerStatus.State)
			}
		} else if triggerStatus.State != "Present" {
			t.Fatalf("Trigger state ResourceVersion for %s with name %s should have been '%s', but was '%s'",
				triggerStatus.Kind, triggerStatus.Name, "Present", triggerStatus.State)
		}
	}
	targetStatuses := crd.Status.Targets
	if len(targetStatuses) != len(targets) {
		t.Fatalf("There is a mismatch between CRD target statuses count and targets count %d != %d", len(targetStatuses), len(targets))
	}
	for idx, targetStatus := range targetStatuses {
		target := &targets[idx]
		if targetStatus.Kind != target.Kind {
			t.Fatalf("Targetstatus with idx %d should have been of kind %s, but was of kind %s", idx, target.Kind, targetStatus.Kind)
		}
		if targetStatus.Name != target.Name {
			t.Fatalf("Targetstatus with idx %d should have had name %s, but had name %s", idx, target.Name, targetStatus.Name)
		}
		expectedTargetStatusState := expectedTargetStatusStates[fmt.Sprintf("%s/%s", targetStatus.Kind, targetStatus.Name)]
		if targetStatus.State != expectedTargetStatusState {
			t.Fatalf("State for target with kind %s and name %s was '%s', but should have been '%s'",
				targetStatus.Kind, targetStatus.Name, targetStatus.State, expectedTargetStatusState)
		}
		targetTriggerStatuses := targetStatus.Triggers
		if !reflect.DeepEqual(targetTriggerStatuses, triggerStatuses) {
			t.Fatalf("Trigger statuses for target with kind %s and name %s was %#v, but should have been %#v",
				targetStatus.Kind, targetStatus.Name, targetTriggerStatuses, triggerStatuses)
		}
	}
	// Validate target rollout annotation
	for _, target := range targets {
		kind := target.Kind
		name := target.Name
		resource := &unstructured.Unstructured{}
		resource.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "apps",
			Kind:    kind,
			Version: "v1",
		})
		err := r.client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, resource)
		if err != nil {
			if errors.IsNotFound(err) {
				continue // Some targets have not been created, so this is ok
			}
			t.Fatalf("Unable to find %s with name %s. The error is %#v", kind, name, err)
		}
		resourceTemplate, found, err := unstructured.NestedMap(resource.Object, "spec", "template")
		annotation := ""
		if err != nil {
			t.Fatalf("When looking for annotations - nesting spec and template, found: %#v, err: %#v", found, err)
		}
		if found {
			targetResourceAnnotations, found, err := unstructured.NestedStringMap(resourceTemplate, "metadata", "annotations")
			if err != nil {
				t.Fatalf("When looking for annotations nesting metadata and annotations, found: %#v, err: %#v", found, err)
			}
			if found {
				annotation = targetResourceAnnotations[rolloutAnnotationName]
			}
		}
		if expectAnnotation == false && annotation != "" {
			t.Fatalf("The %s with name %s was annotated. The annotation is %s=%s", kind, name, rolloutAnnotationName, annotation)
		} else if expectAnnotation == true && annotation == "" {
			t.Fatalf("The %s with name %s was not annotated. The annotation %s should have been present in %#v. The object was %#v",
				kind, name, rolloutAnnotationName, resource, resource.Object)
		}
	}
}

// Remove any rollout annotations from targets in order to prepare for the next test
func removeRolloutAnnotation(t *testing.T, namespace string,
	targets []reloadrestarttriggerv1alpha1.TargetReference,
	r *ReconcileResourceReloadRestartTrigger) {
	// Validate that no targets have been annotated
	for _, target := range targets {
		kind := target.Kind
		name := target.Name
		resource := &unstructured.Unstructured{}
		resource.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "apps",
			Kind:    kind,
			Version: "v1",
		})
		err := r.client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, resource)
		if err != nil {
			if errors.IsNotFound(err) {
				continue // This is ok as we know some targets doesn't exist
			}
			t.Fatalf("Unable to find %s with name %s. The error is %#v", kind, name, err)
		}
		resourceTemplate, found, err := unstructured.NestedMap(resource.Object, "spec", "template")
		if err != nil {
			t.Fatalf("When looking for annotations - nesting spec and template, found: %#v, err: %#v", found, err)
		}
		if found {
			targetResourceAnnotations, found, err := unstructured.NestedStringMap(resourceTemplate, "metadata", "annotations")
			if err != nil {
				t.Fatalf("When looking for annotations nesting metadata and annotations, found: %#v, err: %#v", found, err)
			}
			if found {
				if targetResourceAnnotations[rolloutAnnotationName] != "" {
					delete(targetResourceAnnotations, rolloutAnnotationName)
					err = unstructured.SetNestedStringMap(resourceTemplate, targetResourceAnnotations, "metadata", "annotations")
					if err != nil {
						t.Fatalf("Unable to set the value of the metadata=>annotations object, got error %#v", err)
					}
					err = unstructured.SetNestedMap(resource.Object, resourceTemplate, "spec", "template")
					if err != nil {
						t.Fatalf("Unable to set the value of the spec=>template object %#v, got error %#v", resource, err)
					}
					err = r.client.Update(context.TODO(), resource)
					if err != nil {
						t.Fatalf("Unable to remove the rollout annotation from object %#v", resource.Object)
					}
				}
			}
		}
	}
}
