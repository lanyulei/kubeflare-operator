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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	clusterv1 "kubeflare.com/operator/api/cluster/v1"
)

var _ = Describe("Cluster Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name: resourceName,
		}
		cluster := &clusterv1.Cluster{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind Cluster")
			err := k8sClient.Get(ctx, typeNamespacedName, cluster)
			if err != nil && errors.IsNotFound(err) {
				resource := &clusterv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: resourceName,
					},
					Spec: clusterv1.ClusterSpec{
						Connection: clusterv1.ClusterConnection{
							Kubeconfig: []byte("fake-kubeconfig"),
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			resource := &clusterv1.Cluster{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance Cluster")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &ClusterReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
				StatusReader: fakeClusterStatusReader{
					kubernetesVersion: "v1.30.0",
					nodeCount:         3,
				},
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("checking the synchronized cluster status")
			updated := &clusterv1.Cluster{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, updated)).To(Succeed())
			Expect(updated.Status.KubernetesVersion).To(Equal("v1.30.0"))
			Expect(updated.Status.NodeCount).To(Equal(int32(3)))
		})
	})
})

type fakeClusterStatusReader struct {
	kubernetesVersion string
	nodeCount         int32
}

// fakeClusterStatusReader 避免测试依赖真实的远端 Kubernetes 集群。
func (f fakeClusterStatusReader) ReadClusterStatus(context.Context, []byte) (observedClusterStatus, error) {
	return observedClusterStatus{
		KubernetesVersion: f.kubernetesVersion,
		NodeCount:         f.nodeCount,
	}, nil
}
