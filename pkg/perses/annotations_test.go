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

package perses

import (
	"testing"

	v1 "github.com/istio-ecosystem/sail-operator/api/v1"
	"github.com/istio-ecosystem/sail-operator/pkg/constants"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDashboardsEnabled(t *testing.T) {
	istio := &v1.Istio{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				constants.PersesDashboardsAnnotationKey: constants.PersesDashboardsEnabledValue,
			},
		},
	}
	if !DashboardsEnabled(istio) {
		t.Fatal("expected dashboards to be enabled")
	}

	istio.Annotations[constants.PersesDashboardsAnnotationKey] = "disabled"
	if DashboardsEnabled(istio) {
		t.Fatal("expected dashboards to be disabled")
	}
}

func TestProjectNamespace(t *testing.T) {
	istio := &v1.Istio{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				constants.PersesProjectAnnotationKey: "monitoring",
			},
		},
	}
	ns, err := ProjectNamespace(istio)
	if err != nil {
		t.Fatalf("ProjectNamespace() error = %v", err)
	}
	if ns != "monitoring" {
		t.Fatalf("expected monitoring, got %q", ns)
	}

	istio.Annotations[constants.PersesProjectAnnotationKey] = ""
	if _, err := ProjectNamespace(istio); err == nil {
		t.Fatal("expected error for missing perses-project annotation")
	}
}
