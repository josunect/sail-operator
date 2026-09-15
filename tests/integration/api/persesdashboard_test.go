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

	v1 "github.com/istio-ecosystem/sail-operator/api/v1"
	"github.com/istio-ecosystem/sail-operator/pkg/constants"
	"github.com/istio-ecosystem/sail-operator/pkg/perses"
)

var _ = Describe("Perses dashboard provisioning", Ordered, func() {
	const (
		istioName     = "perses-dashboard-test"
		namespaceName = "perses-dashboard-monitoring"
	)

	ctx := context.Background()
	istioKey := types.NamespacedName{Name: istioName}

	SetDefaultEventuallyTimeout(30 * time.Second)
	SetDefaultEventuallyPollingInterval(time.Second)

	BeforeAll(func() {
		preserve := true
		Expect(k8sClient.Create(ctx, &apiextensionsv1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{Name: perses.PersesDashboardCRD},
			Spec: apiextensionsv1.CustomResourceDefinitionSpec{
				Group: "perses.dev",
				Names: apiextensionsv1.CustomResourceDefinitionNames{
					Kind:     "PersesDashboard",
					ListKind: "PersesDashboardList",
					Plural:   "persesdashboards",
					Singular: "persesdashboard",
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

		Expect(k8sClient.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespaceName}})).To(Succeed())
	})

	AfterAll(func() {
		_ = k8sClient.Delete(ctx, &v1.Istio{ObjectMeta: metav1.ObjectMeta{Name: istioName}})
		_ = k8sClient.Delete(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespaceName}})
	})

	It("creates PersesDashboard resources in the annotated namespace", func() {
		istio := &v1.Istio{
			ObjectMeta: metav1.ObjectMeta{
				Name: istioName,
				Annotations: map[string]string{
					constants.PersesDashboardsAnnotationKey: constants.PersesDashboardsEnabledValue,
					constants.PersesProjectAnnotationKey:    namespaceName,
				},
			},
			Spec: v1.IstioSpec{
				Version:   "v1.30.3",
				Namespace: "istio-system",
			},
		}
		Expect(k8sClient.Create(ctx, istio)).To(Succeed())

		Eventually(func(g Gomega) {
			got := &v1.Istio{}
			g.Expect(k8sClient.Get(ctx, istioKey, got)).To(Succeed())
			cond := got.Status.GetCondition(v1.IstioConditionPersesDashboardsAvailable)
			g.Expect(cond.Status).To(Equal(metav1.ConditionTrue))
			g.Expect(cond.Reason).To(Equal(v1.IstioReasonPersesDashboardsAvailable))
		}).Should(Succeed())

		list := &unstructured.UnstructuredList{}
		list.SetGroupVersionKind(schema.GroupVersionKind{Group: "perses.dev", Version: "v1alpha2", Kind: "PersesDashboardList"})
		Expect(k8sClient.List(ctx, list, client.InNamespace(namespaceName))).To(Succeed())
		Expect(list.Items).To(HaveLen(len(perses.ProductDashboards)))
	})
})
