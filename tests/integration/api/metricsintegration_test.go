//go:build integration

// Copyright Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package integration

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/istio-ecosystem/sail-operator/api/v1alpha1"
	"github.com/istio-ecosystem/sail-operator/pkg/perses"
)

var _ = Describe("MetricsIntegration Perses provisioning", Ordered, func() {
	const (
		integrationName = "test-perses"
		namespaceName   = "metricsintegration-monitoring"
		datasourceName  = perses.DefaultDatasourceName
	)

	ctx := context.Background()
	integrationKey := types.NamespacedName{Name: integrationName}

	SetDefaultEventuallyTimeout(30 * time.Second)
	SetDefaultEventuallyPollingInterval(time.Second)

	BeforeAll(func() {
		preserve := true
		for _, crd := range []struct {
			name   string
			kind   string
			plural string
		}{
			{perses.PersesDatasourceCRD, "PersesDatasource", "persesdatasources"},
			{perses.PersesDashboardCRD, "PersesDashboard", "persesdashboards"},
		} {
			Expect(k8sClient.Create(ctx, &apiextensionsv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{Name: crd.name},
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Group: "perses.dev",
					Names: apiextensionsv1.CustomResourceDefinitionNames{
						Kind:     crd.kind,
						ListKind: crd.kind + "List",
						Plural:   crd.plural,
						Singular: crd.plural,
					},
					Scope: apiextensionsv1.NamespaceScoped,
					Versions: []apiextensionsv1.CustomResourceDefinitionVersion{{
						Name:    "v1alpha2",
						Served:  true,
						Storage: true,
						Schema: &apiextensionsv1.CustomResourceValidation{
							OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
								Type:                   "object",
								XPreserveUnknownFields: &preserve,
							},
						},
					}},
				},
			})).To(Succeed())
		}

		Expect(k8sClient.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespaceName}})).To(Succeed())

		datasource := &unstructured.Unstructured{}
		datasource.SetGroupVersionKind(schema.GroupVersionKind{Group: "perses.dev", Version: "v1alpha2", Kind: "PersesDatasource"})
		datasource.SetName(datasourceName)
		datasource.SetNamespace(namespaceName)
		datasource.Object["spec"] = map[string]interface{}{}
		Expect(k8sClient.Create(ctx, datasource)).To(Succeed())
	})

	AfterAll(func() {
		_ = k8sClient.Delete(ctx, &v1alpha1.MetricsIntegration{ObjectMeta: metav1.ObjectMeta{Name: integrationName}})
		_ = k8sClient.Delete(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespaceName}})
	})

	It("server-side applies datasource fields and provisions dashboards", func() {
		mi := &v1alpha1.MetricsIntegration{
			ObjectMeta: metav1.ObjectMeta{Name: integrationName},
			Spec: v1alpha1.MetricsIntegrationSpec{
				TargetRefs: []v1alpha1.TargetReference{{
					Kind:      "PersesDatasource",
					Name:      datasourceName,
					Namespace: namespaceName,
				}},
				MetricsConfig: v1alpha1.MetricsConfig{
					Type:                   v1alpha1.MetricsTypeUserWorkloadMonitoring,
					UserWorkloadMonitoring: &v1alpha1.UserWorkloadMonitoringConfig{},
				},
			},
		}
		Expect(k8sClient.Create(ctx, mi)).To(Succeed())

		Eventually(func(g Gomega) {
			got := &v1alpha1.MetricsIntegration{}
			g.Expect(k8sClient.Get(ctx, integrationKey, got)).To(Succeed())
			reconciled := got.Status.GetCondition(v1alpha1.MetricsIntegrationConditionReconciled)
			g.Expect(reconciled.Status).To(Equal(metav1.ConditionTrue))
			persesAvailable := got.Status.GetCondition(v1alpha1.MetricsIntegrationConditionPersesAvailable)
			g.Expect(persesAvailable.Status).To(Equal(metav1.ConditionTrue))
		}).Should(Succeed())

		ds := &unstructured.Unstructured{}
		ds.SetGroupVersionKind(schema.GroupVersionKind{Group: "perses.dev", Version: "v1alpha2", Kind: "PersesDatasource"})
		Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: namespaceName, Name: datasourceName}, ds)).To(Succeed())
		Expect(ds.GetOwnerReferences()).To(BeEmpty())

		list := &unstructured.UnstructuredList{}
		list.SetGroupVersionKind(schema.GroupVersionKind{Group: "perses.dev", Version: "v1alpha2", Kind: "PersesDashboardList"})
		Expect(k8sClient.List(ctx, list, client.InNamespace(namespaceName))).To(Succeed())
		Expect(list.Items).To(HaveLen(len(perses.ProductDashboards)))
	})
})
