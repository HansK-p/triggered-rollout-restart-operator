/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
type TriggerReference struct {
	// +kubebuilder:validation:Enum=ConfigMap;Secret
	// Kind is the K8s object kind
	Kind string `json:"kind"`
	// Name is the name of the Secret
	Name string `json:"name"`
}

type TargetReference struct {
	// +kubebuilder:validation:Enum=Deployment;DaemonSet;StatefulSet
	// Kind is the K8s object kind
	Kind string `json:"kind"`
	// Name is the K8s object name
	Name string `json:"name"`
}

type TriggerStatus struct {
	// Kind is the K8s object kind
	Kind string `json:"kind"`
	// Name is the K8s Secret name
	Name string `json:"name"`
	// State is the K8s Secret state
	State string `json:"state,omitempty"`
	// ResourceVersion is the last K8s resourceVersion seen
	ResourceVersion string `json:"resourceVersion"`
}

type TargetStatus struct {
	// Kind is the K8s object kind
	Kind string `json:"kind"`
	// Name is the K8s object name
	Name string `json:"name"`
	// State is target state
	State string `json:"state"`
	// TriggerStatuses is the dependent secrets resourceVersion on the last reload
	Triggers []TriggerStatus `json:"triggers"`
}

// ResourceReloadRestartTriggerSpec defines the desired state of ResourceReloadRestartTrigger
type ResourceReloadRestartTriggerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Secrets is a list of secrets where a change should trigger a reload restart
	// +kubebuilder:validation:MinItems=1
	Triggers []TriggerReference `json:"triggers,omitempty"`

	// +kubebuilder:validation:MinItems=1
	// Targets is a list of targets that will be reloaded when triggered
	Targets []TargetReference `json:"targets,omitempty"`
}

// ResourceReloadRestartTriggerStatus defines the observed state of ResourceReloadRestartTrigger
type ResourceReloadRestartTriggerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Triggers []TriggerStatus `json:"triggers,omitempty"`
	Targets  []TargetStatus  `json:"targets,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ResourceReloadRestartTrigger is the Schema for the resourcereloadrestarttriggers API
type ResourceReloadRestartTrigger struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ResourceReloadRestartTriggerSpec   `json:"spec,omitempty"`
	Status ResourceReloadRestartTriggerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ResourceReloadRestartTriggerList contains a list of ResourceReloadRestartTrigger
type ResourceReloadRestartTriggerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ResourceReloadRestartTrigger `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ResourceReloadRestartTrigger{}, &ResourceReloadRestartTriggerList{})
}
