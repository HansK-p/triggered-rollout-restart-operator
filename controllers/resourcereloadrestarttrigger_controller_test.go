package controllers

import (
	"context"
	"fmt"
	"reflect"
	reloadrestarttriggerv1alpha1 "reload-restart-trigger/api/v1alpha1"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	//v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

const (
	timeout  = time.Second * 10
	duration = time.Second * 10
	interval = time.Millisecond * 250
)

var _ = Describe("ReloadRestartTrigger controller", func() {

	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		crName        = "resourcereloadrestarttrigger-operator"
		image         = "nginx"
		containerName = "nginx-container"
		serviceName   = "nginx-service"
	)
	var (
		triggers  = []reloadrestarttriggerv1alpha1.TriggerReference{}
		targets   = []reloadrestarttriggerv1alpha1.TargetReference{}
		namespace = "ns"
	)
	BeforeEach(func() {
		triggers = []reloadrestarttriggerv1alpha1.TriggerReference{
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
		namespace = fmt.Sprintf("ns-%s", randString(10))

		ctx := context.Background()

		By("By creating a containing namespace")
		nsSpec := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
		Expect(k8sClient.Create(ctx, nsSpec)).Should(Succeed())

		By("Register a ResourceReloadRestartTrigger CRD")
		crd1 := &reloadrestarttriggerv1alpha1.ResourceReloadRestartTrigger{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "reload-restart-trigger.k8s.faith/v1alpha1",
				Kind:       "ResourceReloadRestartTrigger",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      crName,
				Namespace: namespace,
			},
			Spec: reloadrestarttriggerv1alpha1.ResourceReloadRestartTriggerSpec{
				Triggers: triggers,
				Targets:  targets,
			},
		}
		Expect(k8sClient.Create(ctx, crd1)).Should(Succeed())

		By("Deploying a Deployment which will be gracefully restarted by the Controller using annotation changes")
		deployment1 := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "deployment-1",
				Namespace: namespace,
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"deployment": "deployment-1"},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"deployment": "deployment-1"}},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  containerName,
								Image: image,
							},
						},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, deployment1)).Should(Succeed())

		lookupKey := types.NamespacedName{Name: "deployment-1", Namespace: namespace}
		createdDeployment := &appsv1.Deployment{}

		// We'll need to retry getting this newly deployed application, given that creation may not immediately happen.
		Eventually(func() bool {
			err := k8sClient.Get(ctx, lookupKey, createdDeployment)
			return err == nil
		}, timeout, interval).Should(BeTrue())

		By("Deploying a DaemonSet which will be gracefully restarted by the Controller using annotation changes")
		daemonset1 := &appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "daemonset-1",
				Namespace: namespace,
			},
			Spec: appsv1.DaemonSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"daemonset": "daemonset-1"},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"daemonset": "daemonset-1"},
					},
					Spec: corev1.PodSpec{
						NodeSelector: map[string]string{"daemonset": "daemonset-1"},
						Containers: []corev1.Container{
							{
								Name:  containerName,
								Image: image,
							},
						},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, daemonset1)).Should(Succeed())

		lookupKey.Name = "daemonset-1"
		createdDaemonSet := &appsv1.DaemonSet{}

		// We'll need to retry getting this newly deployed application, given that creation may not immediately happen.
		Eventually(func() bool {
			err := k8sClient.Get(ctx, lookupKey, createdDaemonSet)
			return err == nil
		}, timeout, interval).Should(BeTrue())

		By("Deploying a StatefulSet which will be gracefully restarted by the Controller using annotation changes")
		statefulset1 := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "statefulset-1",
				Namespace: namespace,
			},
			Spec: appsv1.StatefulSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"statefulset": "statefulset-1"},
				},
				ServiceName: serviceName,
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"statefulset": "statefulset-1"},
					},
					Spec: corev1.PodSpec{
						NodeSelector: map[string]string{"statefulset": "statefulset-1"},
						Containers: []corev1.Container{
							{
								Name:  containerName,
								Image: image,
							},
						},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, statefulset1)).Should(Succeed())

		lookupKey.Name = "statefulset-1"
		createdStatefulSet := &appsv1.StatefulSet{}

		// We'll need to retry getting this newly deployed application, given that creation may not immediately happen.
		Eventually(func() bool {
			err := k8sClient.Get(ctx, lookupKey, createdStatefulSet)
			return err == nil
		}, timeout, interval).Should(BeTrue())

		By("Deploying a CoonfigMap that will act as trigger")
		configMap1 := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "configmap-1",
				Namespace: namespace,
			},
			Data: map[string]string{
				"file1": "content1",
				"file2": "content2",
			},
		}
		Expect(k8sClient.Create(ctx, configMap1)).Should(Succeed())

		By("Deploying a 2nd CoonfigMap that will act as trigger")
		configMap2 := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "configmap-2",
				Namespace: namespace,
			},
			Data: map[string]string{
				"file11": "content11",
				"file21": "content21",
			},
		}
		Expect(k8sClient.Create(ctx, configMap2)).Should(Succeed())

		By("Deploying a Secret that will act as trigger")
		secret1 := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret-1",
				Namespace: namespace,
			},
			Data: map[string][]byte{
				"secret1": []byte("content1"),
				"secret2": []byte("content2"),
			},
		}
		Expect(k8sClient.Create(ctx, secret1)).Should(Succeed())

		By("By checking that the status is 'NotChecked' for all target deployments, daemonsets and statefulsets before any triggers are changed")
		// ctx := context.Background()

		validateStatusAfterReconcile(namespace, crName, triggers, targets, false, map[string]string{
			"Deployment/deployment-1":               "NotChecked",
			"Deployment/deployment-not-to-be-found": "NotChecked",
			"DaemonSet/daemonset-1":                 "NotChecked",
			"StatefulSet/statefulset-1":             "NotChecked",
		})
	})
	Context("Testing the controller", func() {
		It("Should validate that targets (applications) are gracefully restarted when triggers are changed", func() {
			doInitialTriggerChangeAndTargetAnnotationRemove(namespace, crName, triggers, targets)
		})
		It("Should handle graceful restart of applications when a trigger is added to the CR after initial deployment", func() {
			doInitialTriggerChangeAndTargetAnnotationRemove(namespace, crName, triggers, targets)

			By("By adding yet another configmap to watch to the crd. This should not trigger a rolling upgrade")
			triggers = append(triggers, reloadrestarttriggerv1alpha1.TriggerReference{Kind: "ConfigMap", Name: "configmap-2"})
			updateCRD(namespace, crName, triggers, targets)

			By("Keeping the same Found and NotFound statuses for Deployments, DaemonSets and StatefulSets")
			validateStatusAfterReconcile(namespace, crName, triggers, targets, false, map[string]string{
				"Deployment/deployment-1":               "Found",
				"Deployment/deployment-not-to-be-found": "NotFound",
				"DaemonSet/daemonset-1":                 "Found",
				"StatefulSet/statefulset-1":             "Found",
			})

			By("By adding an annotation to the newly added configmap")
			updateResource(namespace, "ConfigMap", "configmap-2", "before-reconcile-test", "1st")

			By("Validating that the CR has initiated a graceful restart of all Deployments, DaemonSets and StatefulSets by adding an annotation to each of them")
			validateStatusAfterReconcile(namespace, crName, triggers, targets, true, map[string]string{
				"Deployment/deployment-1":               "Found",
				"Deployment/deployment-not-to-be-found": "NotFound",
				"DaemonSet/daemonset-1":                 "Found",
				"StatefulSet/statefulset-1":             "Found",
			})
		})

		It("Should handle graceful restart of applications when a trigger is removed from the CR after initial deployment", func() {
			doInitialTriggerChangeAndTargetAnnotationRemove(namespace, crName, triggers, targets)
			newTriggers := []reloadrestarttriggerv1alpha1.TriggerReference{}

			By("By creating a new trigger list only with triggers we know can be found")
			for _, trigger := range triggers {
				if !strings.Contains(trigger.Name, "not-to-be-found") {
					newTriggers = append(newTriggers, trigger)
				}
			}
			if len(triggers) == len(newTriggers) {
				Expect(fmt.Errorf("Unable to reduce number of triggers when removing triggers not in use, %d == %d", len(triggers), len(newTriggers))).Should(Succeed())
			}
			triggers = newTriggers
			By("By updating the CR with the new trigger list container only triggers we know can be found")
			updateCRD(namespace, crName, triggers, targets)

			By("Validating that the CR did not restart Deployments, DaemonSets and StatefulSets when triggers were removed")
			validateStatusAfterReconcile(namespace, crName, triggers, targets, false, map[string]string{
				"Deployment/deployment-1":               "Found",
				"Deployment/deployment-not-to-be-found": "NotFound",
				"DaemonSet/daemonset-1":                 "Found",
				"StatefulSet/statefulset-1":             "Found",
			})
		})
		It("Should handle graceful restart of applications when a new target is added to the CR after initial deployment", func() {
			doInitialTriggerChangeAndTargetAnnotationRemove(namespace, crName, triggers, targets)
			newTargets := []reloadrestarttriggerv1alpha1.TargetReference{}

			By("Remove targets which doesn't exist")
			for _, target := range targets {
				if !strings.Contains(target.Name, "not-to-be-found") {
					newTargets = append(newTargets, target)
				}
			}
			if len(targets) == len(newTargets) {
				Expect(fmt.Errorf("Unable to reduce number of targets when removing targets that can't be found, %d == %d", len(targets), len(newTargets))).Should(Succeed())
			}
			targets = newTargets

			By("Updatinge the CR with the new reduced list of targets")
			updateCRD(namespace, crName, triggers, targets)

			By("Validating that the reduction of targets didn't trigger a gracefull restart of targets, that is Deployments, DaemonSets and StatefulSets")
			validateStatusAfterReconcile(namespace, crName, triggers, targets, false, map[string]string{
				"Deployment/deployment-1":   "Found",
				"DaemonSet/daemonset-1":     "Found",
				"StatefulSet/statefulset-1": "Found",
			})

			By("Updating a annotation on one trigger to initiate a graceful restart of Deployments, DaemonSets and StatefulSets")
			updateResource(namespace, "Secret", "secret-1", "before-reconcile-test", "3dorsomething")

			By("Validating that the remaining Deployments, DaemonSets and StatefulSets were restarted by adding an annotation")
			validateStatusAfterReconcile(namespace, crName, triggers, targets, true, map[string]string{
				"Deployment/deployment-1":   "Found",
				"DaemonSet/daemonset-1":     "Found",
				"StatefulSet/statefulset-1": "Found",
			})
		})
	})
})

func doInitialTriggerChangeAndTargetAnnotationRemove(namespace, crName string,
	triggers []reloadrestarttriggerv1alpha1.TriggerReference,
	targets []reloadrestarttriggerv1alpha1.TargetReference) {
	By("Change all triggers by adding an annotation to them")
	for _, trigger := range triggers {
		updateResource(namespace, trigger.Kind, trigger.Name, "before-reconcile-test", "2nd")
	}
	By("Checking that an annotation has been applied for all Deployments, DaemonSets and StatefulSets. No target should have the 'NotChecked' status anymore")
	validateStatusAfterReconcile(namespace, crName, triggers, targets, true, map[string]string{
		"Deployment/deployment-1":               "Found",
		"Deployment/deployment-not-to-be-found": "NotFound",
		"DaemonSet/daemonset-1":                 "Found",
		"StatefulSet/statefulset-1":             "Found",
	})

	By("By removing rollout annotations from all targets")
	removeRolloutAnnotation(namespace, targets)

	By("By validating that removal of target trigger annotations made no difference to CR statuses Deployments, DaemonSets and StatefulSets")
	validateStatusAfterReconcile(namespace, crName, triggers, targets, false, map[string]string{
		"Deployment/deployment-1":               "Found",
		"Deployment/deployment-not-to-be-found": "NotFound",
		"DaemonSet/daemonset-1":                 "Found",
		"StatefulSet/statefulset-1":             "Found",
	})
}

func updateResource(namespace, kind, name string, annotationName, annotationValue string) {
	resource := &unstructured.Unstructured{}
	resource.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "",
		Kind:    kind,
		Version: "v1",
	})

	err := k8sClient.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, resource)
	if err != nil {
		if errors.IsNotFound(err) {
			return
		}
		Expect(err).Should(Succeed())
	}
	resource.SetAnnotations(map[string]string{annotationName: annotationValue})
	Expect(k8sClient.Update(context.TODO(), resource)).Should(Succeed())
}

func updateCRD(namespace, name string,
	triggers []reloadrestarttriggerv1alpha1.TriggerReference,
	targets []reloadrestarttriggerv1alpha1.TargetReference) {
	crd := &reloadrestarttriggerv1alpha1.ResourceReloadRestartTrigger{}
	Eventually(func() error {
		err := k8sClient.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, crd)
		if err != nil {
			return err
		}
		crd.Spec.Triggers = triggers
		crd.Spec.Targets = targets
		return k8sClient.Update(context.TODO(), crd)
	}, timeout, interval).Should(Succeed())
}

func validateStatusAfterReconcile(namespace, crName string,
	triggers []reloadrestarttriggerv1alpha1.TriggerReference,
	targets []reloadrestarttriggerv1alpha1.TargetReference,
	expectAnnotation bool,
	expectedTargetStatusStates map[string]string,
) {
	time.Sleep(1000 * time.Millisecond)
	cr := &reloadrestarttriggerv1alpha1.ResourceReloadRestartTrigger{}
	Expect(k8sClient.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: crName}, cr)).Should(Succeed())

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
		err := k8sClient.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, resource)
		if err == nil {
			triggerResourceVersionMap[fmt.Sprintf("%s/%s", kind, name)] = resource.GetResourceVersion()
		} else if errors.IsNotFound(err) {
			triggerResourceVersionMap[fmt.Sprintf("%s/%s", kind, name)] = ""
		} else {
			Expect(err).Should(Succeed())
		}
	}

	// Validate that crd status fields have been updated correctly
	triggerStatuses := cr.Status.Triggers
	if len(triggerStatuses) != len(triggers) {
		Expect(fmt.Errorf("There is a mismatch between CRD trigger statuses count and triggers count %d != %d", len(triggerStatuses), len(triggers))).Should(Succeed())
	}
	for idx, triggerStatus := range triggerStatuses {
		trigger := &triggers[idx]
		if triggerStatus.Kind != trigger.Kind {
			Expect(fmt.Errorf("Triggerstatus with idx %d should have been of kind %s, but was of kind %s", idx, trigger.Kind, triggerStatus.Kind)).Should(Succeed())
		}
		if triggerStatus.Name != trigger.Name {
			Expect(fmt.Errorf("Triggerstatus with idx %d should have had name %s, but had name %s", idx, trigger.Name, triggerStatus.Name)).Should(Succeed())
		}
		expectedResourceVersion := triggerResourceVersionMap[fmt.Sprintf("%s/%s", triggerStatus.Kind, triggerStatus.Name)]
		if expectedResourceVersion != triggerStatus.ResourceVersion {
			Expect(fmt.Errorf("Trigger status ResourceVersion for %s with name %s should have been '%s', but was '%s'",
				triggerStatus.Kind, triggerStatus.Name, expectedResourceVersion, triggerStatus.ResourceVersion)).Should(Succeed())
		}
		if expectedResourceVersion == "" {
			if triggerStatus.State != "NotFound" {
				Expect(fmt.Errorf("Trigger status ResourceVersion for %s with name %s should have been '%s', but was '%s'",
					triggerStatus.Kind, triggerStatus.Name, "NotFound", triggerStatus.State)).Should(Succeed())
			}
		} else if triggerStatus.State != "Found" {
			Expect(fmt.Errorf("Trigger state ResourceVersion for %s with name %s should have been '%s', but was '%s'",
				triggerStatus.Kind, triggerStatus.Name, "Found", triggerStatus.State)).Should(Succeed())
		}
	}
	targetStatuses := cr.Status.Targets
	if len(targetStatuses) != len(targets) {
		Expect(fmt.Errorf("There is a mismatch between CRD target statuses count and targets count %d != %d", len(targetStatuses), len(targets))).Should(Succeed())
	}
	for idx, targetStatus := range targetStatuses {
		target := &targets[idx]
		if targetStatus.Kind != target.Kind {
			Expect(fmt.Errorf("Targetstatus with idx %d should have been of kind %s, but was of kind %s", idx, target.Kind, targetStatus.Kind)).Should(Succeed())
		}
		if targetStatus.Name != target.Name {
			Expect(fmt.Errorf("Targetstatus with idx %d should have had name %s, but had name %s", idx, target.Name, targetStatus.Name)).Should(Succeed())
		}
		expectedTargetStatusState := expectedTargetStatusStates[fmt.Sprintf("%s/%s", targetStatus.Kind, targetStatus.Name)]
		if targetStatus.State != expectedTargetStatusState {
			Expect(fmt.Errorf("State for target with kind %s and name %s was '%s', but should have been '%s'",
				targetStatus.Kind, targetStatus.Name, targetStatus.State, expectedTargetStatusState)).Should(Succeed())
		}
		targetTriggerStatuses := targetStatus.Triggers
		if !reflect.DeepEqual(targetTriggerStatuses, triggerStatuses) {
			Expect(fmt.Errorf("Trigger statuses for target with kind %s and name %s was %#v, but should have been %#v",
				targetStatus.Kind, targetStatus.Name, targetTriggerStatuses, triggerStatuses)).Should(Succeed())
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
		err := k8sClient.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, resource)
		if err != nil {
			if errors.IsNotFound(err) {
				continue // Some targets have not been created, so this is ok
			}
			Expect(fmt.Errorf("Unable to find %s with name %s. The error is %#v", kind, name, err)).Should(Succeed())
		}
		resourceTemplate, deploymentFound, err := unstructured.NestedMap(resource.Object, "spec", "template")
		if err != nil {
			Expect(fmt.Errorf("When looking for annotations - nesting spec and template, found: %#v, err: %#v", deploymentFound, err)).Should(Succeed())
		}
		if deploymentFound {
			targetResourceAnnotations, annotationsFound, err := unstructured.NestedStringMap(resourceTemplate, "metadata", "annotations")
			if err != nil {
				Expect(fmt.Errorf("When looking for annotations nesting metadata and annotations, found: %#v, err: %#v", annotationsFound, err)).Should(Succeed())
			}
			annotation := ""
			if annotationsFound {
				annotation = targetResourceAnnotations[rolloutAnnotationName]
				Expect(fmt.Errorf("Annotation for the target %s of kind %s is %s", name, kind, annotation))
			}
			if !expectAnnotation && annotation != "" {
				Expect(fmt.Errorf("The %s with name %s was annotated. The annotation is %s=%s", kind, name, rolloutAnnotationName, annotation)).Should(Succeed())
			} else if expectAnnotation && annotation == "" {
				Expect(fmt.Errorf("The %s with name %s was not annotated. The annotation %s should have been present in %#v. The object was %#v",
					kind, name, rolloutAnnotationName, resource, resource.Object)).Should(Succeed())
			}
		}
	}
}

// Remove any rollout annotations from targets in order to prepare for the next test
func removeRolloutAnnotation(namespace string,
	targets []reloadrestarttriggerv1alpha1.TargetReference) {
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
		err := k8sClient.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, resource)
		if err != nil {
			if errors.IsNotFound(err) {
				continue // This is ok as we know some targets doesn't exist
			}
			Expect(fmt.Errorf("Unable to find %s with name %s. The error is %#v", kind, name, err)).Should(Succeed())
		}
		resourceTemplate, found, err := unstructured.NestedMap(resource.Object, "spec", "template")
		if err != nil {
			Expect(fmt.Errorf("When looking for annotations - nesting spec and template, found: %#v, err: %#v", found, err)).Should(Succeed())
		}
		if found {
			targetResourceAnnotations, found, err := unstructured.NestedStringMap(resourceTemplate, "metadata", "annotations")
			if err != nil {
				Expect(fmt.Errorf("When looking for annotations nesting metadata and annotations, found: %#v, err: %#v", found, err)).Should(Succeed())
			}
			if found {
				if targetResourceAnnotations[rolloutAnnotationName] != "" {
					delete(targetResourceAnnotations, rolloutAnnotationName)
					err = unstructured.SetNestedStringMap(resourceTemplate, targetResourceAnnotations, "metadata", "annotations")
					if err != nil {
						Expect(fmt.Errorf("Unable to set the value of the metadata=>annotations object, got error %#v", err)).Should(Succeed())
					}
					err = unstructured.SetNestedMap(resource.Object, resourceTemplate, "spec", "template")
					if err != nil {
						Expect(fmt.Errorf("Unable to set the value of the spec=>template object %#v, got error %#v", resource, err)).Should(Succeed())
					}
					err = k8sClient.Update(context.TODO(), resource)
					if err != nil {
						Expect(fmt.Errorf("Unable to remove the rollout annotation from object %#v", resource.Object)).Should(Succeed())
					}
				}
			}
		}
	}
}
