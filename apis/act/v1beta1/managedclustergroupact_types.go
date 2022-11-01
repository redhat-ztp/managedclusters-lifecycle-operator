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
	common "github.com/redhat-ztp/managedclusters-lifecycle-operator/apis/common/v1beta1"
	actionv1beta1 "github.com/stolostron/cluster-lifecycle-api/action/v1beta1"
	viewv1beta1 "github.com/stolostron/cluster-lifecycle-api/view/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// Action
type Action struct {
	Name       string                   `json:"name"`
	ActionSpec actionv1beta1.ActionSpec `json:"spec"`
}

// View
type View struct {
	Name     string               `json:"name"`
	ViewSpec viewv1beta1.ViewSpec `json:"spec"`
}

// ManagedClusterGroupActSpec defines the desired state of ManagedClusterGroupAct
type ManagedClusterGroupActSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	// +optional
	Placement common.GenericPlacementReference `json:"placement,omitempty"`
	// +optional
	common.GenericPlacementFields `json:",inline"`
	// List of ManagedClusterActions to be applied on the selected clusters
	// +optional
	Actions []Action `json:"actions,omitempty"`
	// List of ManagedClusterViews to be applied on the selected clusters
	// +optional
	Views []View `json:"views,omitempty"`
}

// ClusterActState indicate the selected clusters act status
type ClusterActState struct {
	// ManagedCluster Name
	Name string `json:"name"`
	// ManagedClusterActions Status
	//ActionsStatus map[string]string `json:"actionsStatus,omitempty"`
	ActionsStatus string `json:"actionsStatus,omitempty"`
	// ManagedClusterViews Status
	ViewsStatus string `json:"viewsStatus,omitempty"`
}

// ManagedClusterGroupActStatus defines the observed state of ManagedClusterGroupAct
type ManagedClusterGroupActStatus struct {
	//Actions applied to the selected clusters
	AppliedActions string `json:"appliedActions,omitempty"`
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
