/*
Copyright (C) 2022-2024 ApeCloud Co., Ltd

This file is part of KubeBlocks project

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package instanceset

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	workloads "github.com/apecloud/kubeblocks/apis/workloads/v1alpha1"
	"github.com/apecloud/kubeblocks/pkg/controller/builder"
	"github.com/apecloud/kubeblocks/pkg/controller/kubebuilderx"
)

var _ = Describe("status reconciler test", func() {
	BeforeEach(func() {
		its = builder.NewInstanceSetBuilder(namespace, name).
			SetUID(uid).
			SetReplicas(3).
			AddMatchLabelsInMap(selectors).
			SetTemplate(template).
			SetVolumeClaimTemplates(volumeClaimTemplates...).
			SetMinReadySeconds(minReadySeconds).
			SetRoles(roles).
			GetObject()
		priorityMap = ComposeRolePriorityMap(its.Spec.Roles)
	})

	Context("PreCondition & Reconcile", func() {
		It("should work well", func() {
			By("PreCondition")
			its.Generation = 1
			tree := kubebuilderx.NewObjectTree()
			tree.SetRoot(its)

			By("prepare current tree")
			replicas := int32(7)
			its.Spec.Replicas = &replicas
			its.Spec.PodManagementPolicy = appsv1.ParallelPodManagement
			nameHello := "hello"
			instanceHello := workloads.InstanceTemplate{
				Name: nameHello,
			}
			its.Spec.Instances = append(its.Spec.Instances, instanceHello)
			generateNameFoo := "foo"
			replicasFoo := int32(2)
			instanceFoo := workloads.InstanceTemplate{
				Name:     generateNameFoo,
				Replicas: &replicasFoo,
			}
			its.Spec.Instances = append(its.Spec.Instances, instanceFoo)

			// prepare for update
			By("fix meta")
			reconciler = NewFixMetaReconciler()
			newTree, err := reconciler.Reconcile(tree)
			Expect(err).Should(BeNil())

			By("update revisions")
			reconciler = NewRevisionUpdateReconciler()
			newTree, err = reconciler.Reconcile(newTree)
			Expect(err).Should(BeNil())

			By("assistant object")
			reconciler = NewAssistantObjectReconciler()
			newTree, err = reconciler.Reconcile(newTree)
			Expect(err).Should(BeNil())

			By("replicas alignment")
			reconciler = NewReplicasAlignmentReconciler()
			newTree, err = reconciler.Reconcile(newTree)
			Expect(err).Should(BeNil())

			By("all pods are not ready")
			reconciler = NewStatusReconciler()
			Expect(reconciler.PreCondition(newTree)).Should(Equal(kubebuilderx.ResultSatisfied))
			_, err = reconciler.Reconcile(newTree)
			Expect(err).Should(BeNil())
			Expect(its.Status.Replicas).Should(BeEquivalentTo(0))
			Expect(its.Status.ReadyReplicas).Should(BeEquivalentTo(0))
			Expect(its.Status.AvailableReplicas).Should(BeEquivalentTo(0))
			Expect(its.Status.UpdatedReplicas).Should(BeEquivalentTo(0))
			Expect(its.Status.CurrentReplicas).Should(BeEquivalentTo(0))
			Expect(its.Status.CurrentRevisions).Should(HaveLen(0))
			Expect(its.Status.CurrentGeneration).Should(BeEquivalentTo(its.Generation))

			By("make all pods ready with old revision")
			condition := corev1.PodCondition{
				Type:               corev1.PodReady,
				Status:             corev1.ConditionTrue,
				LastTransitionTime: metav1.NewTime(time.Now().Add(-1 * minReadySeconds * time.Second)),
			}
			makePodAvailableWithOldRevision := func(pod *corev1.Pod, revision string) {
				pod.Labels[appsv1.ControllerRevisionHashLabelKey] = revision
				pod.Status.Phase = corev1.PodRunning
				pod.Status.Conditions = append(pod.Status.Conditions, condition)
			}
			pods := newTree.List(&corev1.Pod{})
			for _, object := range pods {
				pod, ok := object.(*corev1.Pod)
				Expect(ok).Should(BeTrue())
				makePodAvailableWithOldRevision(pod, "old-revision")
			}
			_, err = reconciler.Reconcile(newTree)
			Expect(err).Should(BeNil())
			Expect(its.Status.Replicas).Should(BeEquivalentTo(replicas))
			Expect(its.Status.ReadyReplicas).Should(BeEquivalentTo(replicas))
			Expect(its.Status.AvailableReplicas).Should(BeEquivalentTo(replicas))
			Expect(its.Status.UpdatedReplicas).Should(BeEquivalentTo(0))
			Expect(its.Status.CurrentReplicas).Should(BeEquivalentTo(replicas))
			Expect(its.Status.CurrentRevisions).Should(HaveLen(0))
			Expect(its.Status.CurrentGeneration).Should(BeEquivalentTo(its.Generation))

			By("make all pods available with latest revision")
			updateRevisions, err := getUpdateRevisions(its.Status.UpdateRevisions)
			Expect(err).Should(BeNil())
			for _, object := range pods {
				pod, ok := object.(*corev1.Pod)
				Expect(ok).Should(BeTrue())
				makePodAvailableWithOldRevision(pod, updateRevisions[pod.Name])
			}
			_, err = reconciler.Reconcile(newTree)
			Expect(err).Should(BeNil())
			Expect(its.Status.Replicas).Should(BeEquivalentTo(replicas))
			Expect(its.Status.ReadyReplicas).Should(BeEquivalentTo(replicas))
			Expect(its.Status.AvailableReplicas).Should(BeEquivalentTo(replicas))
			Expect(its.Status.UpdatedReplicas).Should(BeEquivalentTo(replicas))
			Expect(its.Status.CurrentReplicas).Should(BeEquivalentTo(replicas))
			Expect(its.Status.CurrentRevisions).Should(Equal(its.Status.UpdateRevisions))
			Expect(its.Status.CurrentGeneration).Should(BeEquivalentTo(its.Generation))
		})
	})

	Context("setMembersStatus function", func() {
		It("should work well", func() {
			pods := []corev1.Pod{
				*builder.NewPodBuilder(namespace, "pod-0").AddLabels(RoleLabelKey, "follower").GetObject(),
				*builder.NewPodBuilder(namespace, "pod-1").AddLabels(RoleLabelKey, "leader").GetObject(),
				*builder.NewPodBuilder(namespace, "pod-2").AddLabels(RoleLabelKey, "follower").GetObject(),
			}
			readyCondition := corev1.PodCondition{
				Type:   corev1.PodReady,
				Status: corev1.ConditionTrue,
			}
			pods[0].Status.Conditions = append(pods[0].Status.Conditions, readyCondition)
			pods[1].Status.Conditions = append(pods[1].Status.Conditions, readyCondition)
			oldMembersStatus := []workloads.MemberStatus{
				{
					PodName:     "pod-0",
					ReplicaRole: &workloads.ReplicaRole{Name: "leader"},
				},
				{
					PodName:     "pod-1",
					ReplicaRole: &workloads.ReplicaRole{Name: "follower"},
				},
				{
					PodName:     "pod-2",
					ReplicaRole: &workloads.ReplicaRole{Name: "follower"},
				},
			}
			replicas := int32(3)
			its.Spec.Replicas = &replicas
			its.Status.MembersStatus = oldMembersStatus
			setMembersStatus(its, &pods)

			Expect(its.Status.MembersStatus).Should(HaveLen(2))
			Expect(its.Status.MembersStatus[0].PodName).Should(Equal("pod-1"))
			Expect(its.Status.MembersStatus[0].ReplicaRole.Name).Should(Equal("leader"))
			Expect(its.Status.MembersStatus[1].PodName).Should(Equal("pod-0"))
			Expect(its.Status.MembersStatus[1].ReplicaRole.Name).Should(Equal("follower"))
		})
	})

	Context("sortMembersStatus function", func() {
		It("should work well", func() {
			// 2(learner)->1(learner)->4(logger)->0(follower)->3(leader)
			membersStatus := []workloads.MemberStatus{
				{
					PodName:     "pod-0",
					ReplicaRole: &workloads.ReplicaRole{Name: "follower"},
				},
				{
					PodName:     "pod-1",
					ReplicaRole: &workloads.ReplicaRole{Name: "learner"},
				},
				{
					PodName:     "pod-2",
					ReplicaRole: &workloads.ReplicaRole{Name: "learner"},
				},
				{
					PodName:     "pod-3",
					ReplicaRole: &workloads.ReplicaRole{Name: "leader"},
				},
				{
					PodName:     "pod-4",
					ReplicaRole: &workloads.ReplicaRole{Name: "logger"},
				},
			}
			expectedOrder := []string{"pod-3", "pod-0", "pod-4", "pod-1", "pod-2"}

			sortMembersStatus(membersStatus, priorityMap)
			for i, status := range membersStatus {
				Expect(status.PodName).Should(Equal(expectedOrder[i]))
			}
		})
	})
})