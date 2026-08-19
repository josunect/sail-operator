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
	"context"
	"fmt"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CRDsAvailable reports whether perses.dev PersesDatasource and PersesDashboard CRDs exist.
func CRDsAvailable(ctx context.Context, cl client.Client) (bool, error) {
	for _, name := range []string{PersesDatasourceCRD, PersesDashboardCRD} {
		crd := &apiextensionsv1.CustomResourceDefinition{}
		if err := cl.Get(ctx, types.NamespacedName{Name: name}, crd); err != nil {
			if client.IgnoreNotFound(err) == nil {
				return false, nil
			}
			return false, fmt.Errorf("get CRD %s: %w", name, err)
		}
	}
	return true, nil
}
