package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

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
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	Triggers []TriggerStatus `json:"triggers,omitempty"`
        Targets []TargetStatus `json:"targets,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ResourceReloadRestartTrigger is the Schema for the resourcereloadrestarttriggers API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=resourcereloadrestarttriggers,scope=Namespaced
type ResourceReloadRestartTrigger struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ResourceReloadRestartTriggerSpec   `json:"spec,omitempty"`
	Status ResourceReloadRestartTriggerStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ResourceReloadRestartTriggerList contains a list of ResourceReloadRestartTrigger
type ResourceReloadRestartTriggerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ResourceReloadRestartTrigger `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ResourceReloadRestartTrigger{}, &ResourceReloadRestartTriggerList{})
}
