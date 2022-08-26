/*
Copyright 2022.

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

package v1beta1

import (
	actionv1beta1 "github.com/stolostron/cluster-lifecycle-api/action/v1beta1"
	viewv1beta1 "github.com/stolostron/cluster-lifecycle-api/view/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// Generic Cluster Reference
type GenericClusterReference struct {
	// Cluster Name
	Name string `json:"name"`
}

// Generic Placement Fields
type GenericPlacementFields struct {
	// Clusters listed with name will be selected and ignoring other clusterSelectors
	Clusters        []GenericClusterReference `json:"clusters,omitempty"`
	ClusterSelector *metav1.LabelSelector     `json:"clusterSelector,omitempty"`
}

// Label Action define the desire action for labeling the selected managed clusters
type LabelAction struct {
	// AddClusterLabels is a map of key/value pairs labels that will be added to the selected managedClusters.
	// If the label already exist it will be update its value
	AddClusterLabels map[string]string `json:"addClusterLabels,omitempty"`
	// DeleteClusterLabels is a map of key/value pairs labels that will be deleted from the selected managedClusters.
	DeleteClusterLabels map[string]string `json:"deleteClusterLabels,omitempty"`
}

// ManagedClusterLabelAction define the desire state to label the selected managed clusters
type ManagedClusterLabelActionSpec struct {
	// Before starting the act make label action for the selected managed cluster.
	BeforeAct LabelAction `json:"beforeAct,omitempty"`
	// After the act is done make label action for the selected managed cluster.
	AfterAct LabelAction `json:"afterAct,omitempty"`
}

// ManagedClusterGroupActSpec defines the desired state of ManagedClusterGroupAct
type ManagedClusterGroupActSpec struct {
	// Important: Run "make" to regenerate code after modifying this file
	GenericPlacementFields `json:",inline"`
	// List of ManagedClusterActions to be applied on the selected clusters
	// +optional
	Actions []actionv1beta1.ManagedClusterAction `json:"actions,omitempty"`
	// List of ManagedClusterViews to be applied on the selected clusters
	// +optional
	Views []viewv1beta1.ManagedClusterView `json:"views,omitempty"`
	// Label actions to be applied on the selected clusters
	// +optional
	ManagedClusterLabelAction *ManagedClusterLabelActionSpec `json:"managedClusterLabelAction,omitempty"`
}

// ClusterActState indicate the selected clusters act status
type ClusterActState struct {
	// ManagedCluster Name
	Name string `json:"name"`
	// ManagedClusterAction resource  state
	ActionState string `json:"actionState,omitempty"`
	// ManagedClusterView resource  state
	ViewState string `json:"viewState,omitempty"`
}

// ManagedClusterGroupActStatus defines the observed state of ManagedClusterGroupAct
type ManagedClusterGroupActStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster

	//List of the ManagedClusterGroupAct conditions
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// List of the selected managedClusters with act status
	Clusters []*ClusterActState `json:"clusters,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ManagedClusterGroupAct is the Schema for the managedclustergroupacts API
type ManagedClusterGroupAct struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ManagedClusterGroupActSpec   `json:"spec,omitempty"`
	Status ManagedClusterGroupActStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ManagedClusterGroupActList contains a list of ManagedClusterGroupAct
type ManagedClusterGroupActList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ManagedClusterGroupAct `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ManagedClusterGroupAct{}, &ManagedClusterGroupActList{})
}
