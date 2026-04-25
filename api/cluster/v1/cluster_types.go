/*
Copyright 2026.

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

// ClusterSpec defines the desired state of Cluster
type ClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html

	// Provider 表示集群提供方，仅用于描述集群来源或类型。
	// 例如：local、kind、eks、ack、tke、自建集群等。
	// +optional
	Provider string `json:"provider,omitempty"`

	// Config 表示集群的自定义配置内容。
	// 该字段会在 CRD 中生成为 type: string, format: byte。
	// +optional
	Config []byte `json:"config,omitempty"`

	// Connection 表示连接目标集群所需的信息。
	// +optional
	Connection ClusterConnection `json:"connection,omitempty"`

	// Enable 表示是否启用该集群。
	// 使用指针是为了区分未设置和显式设置为 false。
	// +optional
	Enable *bool `json:"enable,omitempty"`

	// ExternalKubeAPIEnabled 表示是否对外暴露 Kubernetes API。
	// +optional
	ExternalKubeAPIEnabled bool `json:"externalKubeAPIEnabled,omitempty"`

	// JoinFederation 表示该集群是否加入联邦管理。
	// +optional
	JoinFederation bool `json:"joinFederation,omitempty"`
}

// ClusterConnection 定义连接目标集群所需的信息。
type ClusterConnection struct {
	// Type 表示连接方式。
	// 例如：direct 表示直接连接，proxy 表示代理连接。
	// +optional
	Type string `json:"type,omitempty"`

	// Kubeconfig 表示连接目标集群 API Server 的 kubeconfig 内容。
	// 该字段会在 CRD 中生成为 type: string, format: byte。
	// +optional
	Kubeconfig []byte `json:"kubeconfig,omitempty"`

	// KubernetesAPIEndpoint 表示目标集群 Kubernetes API Server 地址。
	// 例如：https://10.10.0.1:6443。
	// +optional
	KubernetesAPIEndpoint string `json:"kubernetesAPIEndpoint,omitempty"`

	// KubernetesAPIServerPort 表示 Kubernetes API Server 代理转发端口。
	// 通常仅在代理连接模式下使用。
	// +optional
	KubernetesAPIServerPort int32 `json:"kubernetesAPIServerPort,omitempty"`

	// ExternalKubernetesAPIEndpoint 表示对外暴露的 Kubernetes API Server 地址。
	// +optional
	ExternalKubernetesAPIEndpoint string `json:"externalKubernetesAPIEndpoint,omitempty"`

	// Token 表示集群代理或组件连接控制面时使用的令牌。
	// +optional
	Token string `json:"token,omitempty"`
}

// ClusterStatus defines the observed state of Cluster.
type ClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// conditions represent the current state of the Cluster resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	// Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Conditions 表示集群当前状态的最新观测结果。
	// +optional
	Conditions []ClusterCondition `json:"conditions,omitempty"`

	// Configz 表示集群内各组件是否启用。
	// key 为组件名称，value 表示是否启用。
	// +optional
	Configz map[string]bool `json:"configz,omitempty"`

	// KubernetesVersion 表示目标集群的 Kubernetes 版本。
	// +optional
	KubernetesVersion string `json:"kubernetesVersion,omitempty"`

	// NodeCount 表示目标集群节点数量。
	// +optional
	NodeCount int32 `json:"nodeCount,omitempty"`

	// Region 表示集群所在区域。
	// 例如：cn-hangzhou、us-east1。
	// +optional
	Region string `json:"region,omitempty"`

	// UID 表示集群唯一标识。
	// 可以使用 kube-system namespace UID 或其他稳定 ID。
	// +optional
	UID string `json:"uid,omitempty"`

	// Zones 表示集群节点所在的可用区列表。
	// +optional
	Zones []string `json:"zones,omitempty"`
}

// ClusterCondition 表示 Cluster 的一个状态条件。
type ClusterCondition struct {
	// Type 表示条件类型。
	// 例如：Ready、Connected、Healthy。
	Type string `json:"type"`

	// Status 表示条件状态。
	// 可选值通常为 True、False、Unknown。
	Status string `json:"status"`

	// Reason 表示状态变化的原因。
	// +optional
	Reason string `json:"reason,omitempty"`

	// Message 表示状态变化的详细说明。
	// +optional
	Message string `json:"message,omitempty"`

	// LastUpdateTime 表示该条件最后一次更新的时间。
	// +optional
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`

	// LastTransitionTime 表示该条件状态最后一次发生变化的时间。
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="Provider",type=string,JSONPath=".spec.provider"
// +kubebuilder:printcolumn:name="Version",type=string,JSONPath=".status.kubernetesVersion"

// Cluster is the Schema for the clusters API
type Cluster struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of Cluster
	// +required
	Spec ClusterSpec `json:"spec"`

	// status defines the observed state of Cluster
	// +optional
	Status ClusterStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// ClusterList contains a list of Cluster
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []Cluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cluster{}, &ClusterList{})
}
