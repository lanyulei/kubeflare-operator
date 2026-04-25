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

package cluster

import (
	"context"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	clusterv1 "kubeflare.com/operator/api/cluster/v1"
)

type observedClusterStatus struct {
	KubernetesVersion string
	NodeCount         int32
}

const (
	clusterStatusReasonKubeconfigMissing = "KubeconfigMissing"
	clusterStatusReasonStatusSyncFailed  = "StatusSyncFailed"
	clusterStatusReasonStatusSynced      = "StatusSynced"

	// clusterStatusSyncPeriod 定期刷新远端观测状态，避免只在本地对象变化时同步。
	clusterStatusSyncPeriod = 5 * time.Minute
)

// clusterStatusReader 从目标集群读取需要同步到 status 的信息。
type clusterStatusReader interface {
	ReadClusterStatus(ctx context.Context, kubeconfig []byte) (observedClusterStatus, error)
}

type remoteClusterStatusReader struct{}

func (remoteClusterStatusReader) ReadClusterStatus(ctx context.Context, kubeconfig []byte) (observedClusterStatus, error) {
	// kubeconfig 来自 Cluster spec，用于连接被管理的目标集群。
	restConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return observedClusterStatus{}, err
	}

	kubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return observedClusterStatus{}, err
	}

	// 这里只同步最基础的集群观测信息：Kubernetes 版本和节点数量。
	version, err := kubeClient.Discovery().ServerVersion()
	if err != nil {
		return observedClusterStatus{}, err
	}

	nodes, err := kubeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return observedClusterStatus{}, err
	}

	return observedClusterStatus{
		KubernetesVersion: version.GitVersion,
		NodeCount:         int32(len(nodes.Items)),
	}, nil
}

// ClusterReconciler 负责协调 Cluster 资源。
type ClusterReconciler struct {
	crclient.Client
	Scheme *runtime.Scheme
	// StatusReader 允许测试注入假的目标集群读取器，生产环境默认读取真实集群。
	StatusReader clusterStatusReader
}

// +kubebuilder:rbac:groups=cluster.kubeflare.com,resources=clusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cluster.kubeflare.com,resources=clusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cluster.kubeflare.com,resources=clusters/finalizers,verbs=update

// Reconcile 读取目标集群信息，并维护 Cluster 的基础状态字段。
func (r *ClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	cluster := &clusterv1.Cluster{}
	if err := r.Get(ctx, req.NamespacedName, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if len(cluster.Spec.Connection.Kubeconfig) == 0 {
		condition := newClusterCondition(clusterv1.ConditionFalse, clusterStatusReasonKubeconfigMissing,
			"目标集群 kubeconfig 为空，暂时无法同步状态")
		if _, err := r.updateClusterStatus(ctx, cluster, nil, condition); err != nil {
			return ctrl.Result{}, err
		}
		log.Info("Skipped Cluster status sync because kubeconfig is empty", "name", cluster.Name)
		return ctrl.Result{}, nil
	}

	// 未显式注入时，使用真实 Kubernetes 客户端读取目标集群状态。
	statusReader := r.StatusReader
	if statusReader == nil {
		statusReader = remoteClusterStatusReader{}
	}

	observed, err := statusReader.ReadClusterStatus(ctx, cluster.Spec.Connection.Kubeconfig)
	if err != nil {
		condition := newClusterCondition(clusterv1.ConditionFalse, clusterStatusReasonStatusSyncFailed,
			fmt.Sprintf("目标集群状态同步失败：%v", err))
		if _, updateErr := r.updateClusterStatus(ctx, cluster, nil, condition); updateErr != nil {
			return ctrl.Result{}, updateErr
		}
		log.Error(err, "Failed to sync Cluster status", "name", cluster.Name)
		return ctrl.Result{RequeueAfter: clusterStatusSyncPeriod}, nil
	}

	condition := newClusterCondition(clusterv1.ConditionTrue, clusterStatusReasonStatusSynced,
		"目标集群状态已同步")
	updated, err := r.updateClusterStatus(ctx, cluster, &observed, condition)
	if err != nil {
		return ctrl.Result{}, err
	}

	if updated {
		log.Info("Updated Cluster status", "name", cluster.Name,
			"kubernetesVersion", observed.KubernetesVersion, "nodeCount", observed.NodeCount)
	}

	return ctrl.Result{RequeueAfter: clusterStatusSyncPeriod}, nil
}

// updateClusterStatus 只更新 controller 观测到的 status，不修改 spec。
func (r *ClusterReconciler) updateClusterStatus(
	ctx context.Context,
	cluster *clusterv1.Cluster,
	observed *observedClusterStatus,
	condition clusterv1.ClusterCondition,
) (bool, error) {
	changed := false

	if observed != nil {
		if cluster.Status.KubernetesVersion != observed.KubernetesVersion {
			cluster.Status.KubernetesVersion = observed.KubernetesVersion
			changed = true
		}
		if cluster.Status.NodeCount != observed.NodeCount {
			cluster.Status.NodeCount = observed.NodeCount
			changed = true
		}
	}

	if setClusterCondition(&cluster.Status, condition) {
		changed = true
	}

	if !changed {
		return false, nil
	}

	return true, r.Status().Update(ctx, cluster)
}

// newClusterCondition 创建 Ready 条件，具体时间由 setClusterCondition 维护。
func newClusterCondition(status, reason, message string) clusterv1.ClusterCondition {
	return clusterv1.ClusterCondition{
		Type:    clusterv1.ClusterConditionReady,
		Status:  status,
		Reason:  reason,
		Message: message,
	}
}

// setClusterCondition 按 type 更新条件，避免重复追加同类 condition。
func setClusterCondition(status *clusterv1.ClusterStatus, condition clusterv1.ClusterCondition) bool {
	now := metav1.Now()

	for i := range status.Conditions {
		current := status.Conditions[i]
		if current.Type != condition.Type {
			continue
		}

		semanticChanged := current.Status != condition.Status ||
			current.Reason != condition.Reason ||
			current.Message != condition.Message
		if !semanticChanged {
			return false
		}

		condition.LastUpdateTime = now
		if current.Status != condition.Status || current.LastTransitionTime.IsZero() {
			condition.LastTransitionTime = now
		} else {
			condition.LastTransitionTime = current.LastTransitionTime
		}
		status.Conditions[i] = condition
		return true
	}

	condition.LastUpdateTime = now
	condition.LastTransitionTime = now
	status.Conditions = append(status.Conditions, condition)
	return true
}

// SetupWithManager 将 Cluster controller 注册到 Manager。
func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clusterv1.Cluster{}).
		Named("cluster-cluster").
		Complete(r)
}
