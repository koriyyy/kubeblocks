/*
Copyright ApeCloud, Inc.

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

package apps

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	appsv1alpha1 "github.com/apecloud/kubeblocks/apis/apps/v1alpha1"
	intctrlutil "github.com/apecloud/kubeblocks/internal/controllerutil"
	testapps "github.com/apecloud/kubeblocks/internal/testutil/apps"
)

var _ = Describe("Reconcile StorageClass", func() {
	var (
		clusterDefName     = "cluster-def-" + testCtx.GetRandomStr()
		clusterVersionName = "app-versoion-" + testCtx.GetRandomStr()
		clusterName        = "mysql-for-storageclass-" + testCtx.GetRandomStr()
		consensusCompName  = "consensus"
		consensusCompType  = "consensus"
	)

	cleanEnv := func() {
		// must wait until resources deleted and no longer exist before the testcases start,
		// otherwise if later it needs to create some new resource objects with the same name,
		// in race conditions, it will find the existence of old objects, resulting failure to
		// create the new objects.
		By("clean resources")

		// delete cluster(and all dependent sub-resources), clusterversion and clusterdef
		testapps.ClearClusterResources(&testCtx)

		// non-namespaced resources
		testapps.ClearResources(&testCtx, intctrlutil.StorageClassSignature, client.HasLabels{testCtx.TestObjLabelKey})
	}
	BeforeEach(cleanEnv)

	AfterEach(cleanEnv)

	createCluster := func(defaultStorageClassName, storageClassName string) *appsv1alpha1.Cluster {
		cluster := testapps.NewClusterFactory(testCtx.DefaultNamespace, clusterName, clusterDefName, clusterVersionName).
			AddComponent(consensusCompName, consensusCompType).
			AddVolumeClaimTemplate("data", &corev1.PersistentVolumeClaimSpec{
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("2Gi"),
					},
				},
			}).GetObject()
		volumeClaimTemplates := cluster.Spec.ComponentSpecs[0].VolumeClaimTemplates
		volumeClaimTemplates[0].Spec.StorageClassName = &defaultStorageClassName
		volumeClaimTemplates = append(volumeClaimTemplates, appsv1alpha1.ClusterComponentVolumeClaimTemplate{
			Name: "log",
			Spec: &corev1.PersistentVolumeClaimSpec{
				StorageClassName: &storageClassName,
				Resources: corev1.ResourceRequirements{
					Requests: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceStorage: resource.MustParse("2Gi"),
					},
				},
			},
		})
		cluster.Spec.ComponentSpecs[0].VolumeClaimTemplates = volumeClaimTemplates
		cluster.Annotations = map[string]string{
			intctrlutil.StorageClassAnnotationKey: defaultStorageClassName + "," + storageClassName,
		}
		Expect(testCtx.CreateObj(context.Background(), cluster)).Should(Succeed())
		return cluster
	}

	createPVC := func(pvcName, storageClassName string) {
		testapps.NewPersistentVolumeClaimFactory(testCtx.DefaultNamespace, pvcName, clusterName,
			consensusCompName, "data").SetStorage("2Gi").SetStorageClass(storageClassName).Create(&testCtx)
	}

	updateStorageClassAllowVolumeExpansion := func(storageClassName string, allowVolumeExpansion bool) {
		Expect(testapps.GetAndChangeObj(&testCtx, client.ObjectKey{Name: storageClassName}, func(tmpSc *storagev1.StorageClass) {
			tmpSc.AllowVolumeExpansion = &allowVolumeExpansion
		})()).Should(Succeed())

		Eventually(testapps.CheckObj(&testCtx, client.ObjectKey{Name: storageClassName}, func(g Gomega, tmpSc *storagev1.StorageClass) {
			g.Expect(*tmpSc.AllowVolumeExpansion).Should(Equal(allowVolumeExpansion))
		})).Should(Succeed())
	}

	Context("reconcile the Cluster.status.operation.volumeExpandable when StorageClass and PVC changed", func() {
		It("should handle it properly", func() {
			By("init resources")
			vctName1 := "data"
			defaultStorageClassName := "standard-" + testCtx.GetRandomStr()
			storageClassName := "csi-hostpath-sc-" + testCtx.GetRandomStr()
			testapps.CreateConsensusMysqlClusterDef(testCtx, clusterDefName, consensusCompType)
			testapps.CreateConsensusMysqlClusterVersion(testCtx, clusterDefName, clusterVersionName, consensusCompType)
			cluster := createCluster(defaultStorageClassName, storageClassName)
			Expect(testapps.GetAndChangeObjStatus(&testCtx, client.ObjectKeyFromObject(cluster), func(newCluster *appsv1alpha1.Cluster) {
				newCluster.Status.Operations = &appsv1alpha1.Operations{}
				newCluster.Status.Phase = appsv1alpha1.RunningPhase
				newCluster.Status.ObservedGeneration = 1
			})()).Should(Succeed())

			Eventually(testapps.CheckObj(&testCtx, client.ObjectKeyFromObject(cluster), func(g Gomega, newCluster *appsv1alpha1.Cluster) {
				g.Expect(newCluster.Status.Operations != nil).Should(BeTrue())
			})).Should(Succeed())

			By("test without pvc")
			testapps.CreateStorageClass(testCtx, defaultStorageClassName, true)

			By("expect consensus component support volume expansion and volumeClaimTemplateNames is [data]")
			Eventually(testapps.CheckObj(&testCtx, client.ObjectKeyFromObject(cluster), func(g Gomega, newCluster *appsv1alpha1.Cluster) {
				volumeExpandable := newCluster.Status.Operations.VolumeExpandable
				g.Expect(len(volumeExpandable) > 0 && volumeExpandable[0].VolumeClaimTemplateNames[0] == vctName1).Should(BeTrue())
			})).Should(Succeed())

			By("test with pvc")
			testapps.CreateStorageClass(testCtx, storageClassName, false)
			createPVC(fmt.Sprintf("log-%s-%s", clusterName, consensusCompName), storageClassName)
			createPVC(fmt.Sprintf("data-%s-%s", clusterName, consensusCompName), defaultStorageClassName)

			By("expect consensus component support volume expansion and volumeClaimTemplateNames is [data,log]")
			// set storageClass allowVolumeExpansion to true
			updateStorageClassAllowVolumeExpansion(storageClassName, true)

			Eventually(testapps.CheckObj(&testCtx, client.ObjectKeyFromObject(cluster), func(g Gomega, newCluster *appsv1alpha1.Cluster) {
				volumeExpandable := newCluster.Status.Operations.VolumeExpandable
				g.Expect(len(volumeExpandable) > 0 && len(volumeExpandable[0].VolumeClaimTemplateNames) > 1 && volumeExpandable[0].VolumeClaimTemplateNames[1] == "log").Should(BeTrue())
			})).Should(Succeed())

			By("expect consensus component support volume expansion and volumeClaimTemplateNames is [data]")
			// set storageClass allowVolumeExpansion to false
			updateStorageClassAllowVolumeExpansion(storageClassName, false)
			Eventually(testapps.CheckObj(&testCtx, client.ObjectKeyFromObject(cluster), func(g Gomega, newCluster *appsv1alpha1.Cluster) {
				componentVolumeExpandable := newCluster.Status.Operations.VolumeExpandable[0]
				g.Expect(len(componentVolumeExpandable.VolumeClaimTemplateNames) == 1 && componentVolumeExpandable.VolumeClaimTemplateNames[0] == vctName1).Should(BeTrue())
			})).Should(Succeed())

			By("expect consensus component not support volume expansion")
			// set defaultStorageClass allowVolumeExpansion to false
			updateStorageClassAllowVolumeExpansion(defaultStorageClassName, false)
			Eventually(testapps.CheckObj(&testCtx, client.ObjectKeyFromObject(cluster), func(g Gomega, newCluster *appsv1alpha1.Cluster) {
				g.Expect(len(newCluster.Status.Operations.VolumeExpandable) == 0).Should(BeTrue())
			})).Should(Succeed())

			By("expect consensus component support volume expansion and volumeClaimTemplateNames is [data]")
			// set defaultStorageClass allowVolumeExpansion to true
			updateStorageClassAllowVolumeExpansion(defaultStorageClassName, true)
			Eventually(testapps.CheckObj(&testCtx, client.ObjectKeyFromObject(cluster), func(g Gomega, newCluster *appsv1alpha1.Cluster) {
				volumeExpandable := newCluster.Status.Operations.VolumeExpandable
				g.Expect(len(volumeExpandable) > 0 && volumeExpandable[0].VolumeClaimTemplateNames[0] == vctName1).Should(BeTrue())
			})).Should(Succeed())
		})
	})
})