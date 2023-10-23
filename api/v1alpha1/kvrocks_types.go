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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// KVRocksSpec defines the desired state of KVRocks
type KVRocksSpec struct {
	Image           string            `json:"image"`
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
	Type            KVRocksType       `json:"type"`
	KVRocksConfig   map[string]string `json:"kvrocksConfig,omitempty"`
	// RocksDBConfig   map[string]string            `json:"rocksDBConfig,omitempty"`
	Replicas int32 `json:"replicas"`
	// +optional
	Master       uint                         `json:"master"`
	Password     string                       `json:"password"`
	Resources    *corev1.ResourceRequirements `json:"resources"`
	NodeSelector map[string]string            `json:"nodeSelector,omitempty"`
	Toleration   []corev1.Toleration          `json:"toleration,omitempty"`
	Affinity     *corev1.Affinity             `json:"affinity,omitempty"`
	Storage      *KVRocksStorage              `json:"storage,omitempty"`
}

// KVRocksStatus defines the observed state of KVRocks
type KVRocksStatus struct {
	Status    KVRocksStatusType       `json:"status,omitempty"`
	Reason    string                  `json:"reason,omitempty"`
	Version   int                     `json:"version,omitempty"`
	Rebalance bool                    `json:"rebalance,omitempty"`
	Topo      []KVRocksTopoPartitions `json:"topo,omitempty"`
	Shrink    *KVRocksShrinkMsg       `json:"shrink,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.status`
//+kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.reason`
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="CreationTimestamp is a timestamp representing the server time when this object was created. It is not guaranteed to be set in happens-before order across separate operations. Clients may not set this value. It is represented in RFC3339 form and is in UTC."

// KVRocks is the Schema for the kvrocks API
type KVRocks struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KVRocksSpec   `json:"spec,omitempty"`
	Status KVRocksStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KVRocksList contains a list of KVRocks
type KVRocksList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KVRocks `json:"items"`
}

type KVRocksShrinkMsg struct {
	Partition  []int            `json:"partition,omitempty"`
	ReserveMsg map[string][]int `json:"reserveMsg,omitempty"`
}

type KVRocksTopoPartitions struct {
	PartitionName string            `json:"partitionName"`
	Shard         int               `json:"shard"`
	Topology      []KVRocksTopology `json:"topology"`
}

type KVRocksTopology struct {
	Pod      string       `json:"pod"`
	Role     string       `json:"role"`
	NodeId   string       `json:"nodeId"`
	Ip       string       `json:"ip"`
	Port     uint32       `json:"port"`
	Slots    []string     `json:"slots,omitempty"`
	MasterId string       `json:"masterId,omitempty"`
	Migrate  []MigrateMsg `json:"migrate,omitempty"`
	Failover bool         `json:"failover,omitempty"`
}

type MigrateMsg struct {
	Shard int      `json:"shard"`
	Slots []string `json:"slots"`
}

type KVRocksStorage struct {
	Size  resource.Quantity `json:"size"`
	Class string            `json:"class"`
}

type KVRocksType string

const (
	SentinelType KVRocksType = "sentinel"
	StandardType KVRocksType = "standard"
	ClusterType  KVRocksType = "cluster"
)

type KVRocksStatusType string

const (
	StatusNone     KVRocksStatusType = ""
	StatusCreating KVRocksStatusType = "Creating"
	StatusRunning  KVRocksStatusType = "Running"
	StatusFailed   KVRocksStatusType = "Failed"
)

const KVRocksFinalizer = "kvrocks/finalizer"

func init() {
	SchemeBuilder.Register(&KVRocks{}, &KVRocksList{})
}
