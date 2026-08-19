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
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"strings"

	"github.com/istio-ecosystem/sail-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
)

//go:embed dashboards/*.yaml
var dashboardFS embed.FS

// LoadDashboardYAML returns vendored PersesDashboard manifest bytes for a product dashboard.
func LoadDashboardYAML(def DashboardDefinition) ([]byte, error) {
	path := "dashboards/" + def.Filename
	data, err := dashboardFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read dashboard %s: %w", def.Filename, err)
	}
	return data, nil
}

// ParseDashboard unmarshals a PersesDashboard YAML manifest.
func ParseDashboard(data []byte) (*unstructured.Unstructured, error) {
	obj := &unstructured.Unstructured{}
	decoder := utilyaml.NewYAMLOrJSONDecoder(bytes.NewReader(data), 4096)
	if err := decoder.Decode(obj); err != nil {
		return nil, fmt.Errorf("decode dashboard: %w", err)
	}
	return obj, nil
}

// RewriteDatasourceName replaces datasource references in dashboard YAML with the configured name.
func RewriteDatasourceName(data []byte, datasourceName string) ([]byte, error) {
	// Productized dashboards from community-mixins reference prometheus-datasource by default.
	replaced := bytes.ReplaceAll(data, []byte(v1alpha1.DefaultPersesDatasourceName), []byte(datasourceName))
	return replaced, nil
}

// ListBundledDashboardFiles returns dashboard filenames present in the embed FS.
func ListBundledDashboardFiles() ([]string, error) {
	var names []string
	err := fs.WalkDir(dashboardFS, "dashboards", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".yaml") {
			names = append(names, strings.TrimPrefix(path, "dashboards/"))
		}
		return nil
	})
	return names, err
}
