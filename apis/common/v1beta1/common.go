//+kubebuilder:object:generate=true
package common

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Generic Reference
type GenericPlacementReference struct {
	// kind must be either placement or placementRule
	Kind string `json:"kind"`
	// Name
	Name string `json:"name"`
	// Namespace
	Namespace string `json:"namespace"`
}

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
