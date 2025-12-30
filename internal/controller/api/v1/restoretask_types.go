/*
Copyright 2025 404LifeFound.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
type Process int

type Phase string

var (
	ProcessTrue  Process = 1
	ProcessFalse Process = 0

	PhaseGetES                  Phase = "get_es"
	PhaseCreateESNodeSet        Phase = "create_es_node_set"
	PhaseIncreaseNodeSetStorage Phase = "increase_node_set_storage"
	PhaseCheckStatefulsetStatus Phase = "check_sts_status"
	PhaseProcessingRestoring    Phase = "processing_restoring"
	PhaseComplete               Phase = "complete"
)

// RestoreTaskSpec defines the desired state of RestoreTask
type RestoreTaskSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html

	TaskId           string           `json:"taskId"`
	NodeName         string           `json:"nodeName"`
	StoreSize        string           `json:"storeSize"`
	Snapshot         SnapshotRef      `json:"snapshot"`
	Indices          []string         `json:"indices"`
	ElasticsearchRef ElasticsearchRef `json:"elasticsearchRef"`
}

type SnapshotRef struct {
	Repository string `json:"repository"`
	Snapshot   string `json:"snapshot"`
}

type ElasticsearchRef struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

// RestoreTaskStatus defines the observed state of RestoreTask.
type RestoreTaskStatus struct {
	Reason     string             `json:"reason"`
	StartAt    *metav1.Time       `json:"start_at"`
	FinishedAt *metav1.Time       `json:"finished_at"`
	Status     string             `json:"status"`
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// RestoreTask is the Schema for the restoretasks API
type RestoreTask struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of RestoreTask
	// +required
	Spec RestoreTaskSpec `json:"spec"`

	// status defines the observed state of RestoreTask
	// +optional
	Status RestoreTaskStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// RestoreTaskList contains a list of RestoreTask
type RestoreTaskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []RestoreTask `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RestoreTask{}, &RestoreTaskList{})
}
