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
	"fmt"
	"strings"

	v1 "github.com/istio-ecosystem/sail-operator/api/v1"
	"github.com/istio-ecosystem/sail-operator/pkg/constants"
)

// DashboardsEnabled reports whether Perses dashboard provisioning is enabled on the Istio CR.
func DashboardsEnabled(istio *v1.Istio) bool {
	return istio.GetAnnotations()[constants.PersesDashboardsAnnotationKey] == constants.PersesDashboardsEnabledValue
}

// ProjectNamespace returns the target namespace from the perses-project annotation.
func ProjectNamespace(istio *v1.Istio) (string, error) {
	project := strings.TrimSpace(istio.GetAnnotations()[constants.PersesProjectAnnotationKey])
	if project == "" {
		return "", fmt.Errorf("annotation %s is required when %s is enabled",
			constants.PersesProjectAnnotationKey, constants.PersesDashboardsAnnotationKey)
	}
	return project, nil
}
