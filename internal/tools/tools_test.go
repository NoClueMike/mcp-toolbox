// Copyright 2025 Google LLC
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

package tools_test

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
)

func TestGetMcpManifestMetadata(t *testing.T) {
	trueVal := true
	falseVal := false

	authServices := []parameters.ParamAuthService{
		{
			Name:  "my-google-auth-service",
			Field: "auth_field",
		},
		{
			Name:  "other-auth-service",
			Field: "other_auth_field",
		}}
	tcs := []struct {
		desc            string
		name            string
		description     string
		authInvoke      []string
		params          parameters.Parameters
		annotations     *tools.ToolAnnotations
		wantMetadata    map[string]any
		wantAnnotations []byte
	}{
		{
			desc:         "basic manifest without metadata",
			name:         "basic",
			description:  "foo bar",
			authInvoke:   []string{},
			params:       parameters.Parameters{parameters.NewStringParameter("string-param", "string parameter")},
			annotations:  nil,
			wantMetadata: nil,
		},
		{
			desc:            "basic manifest without metadata with annotations",
			name:            "basic",
			description:     "foo bar",
			authInvoke:      []string{},
			params:          parameters.Parameters{parameters.NewStringParameter("string-param", "string parameter")},
			annotations:     &tools.ToolAnnotations{ReadOnlyHint: &trueVal, DestructiveHint: &falseVal},
			wantMetadata:    nil,
			wantAnnotations: []byte(`{"destructiveHint":false,"readOnlyHint":true}`),
		},
		{
			desc:         "with auth invoke metadata",
			name:         "basic",
			description:  "foo bar",
			authInvoke:   []string{"auth1", "auth2"},
			params:       parameters.Parameters{parameters.NewStringParameter("string-param", "string parameter")},
			annotations:  nil,
			wantMetadata: map[string]any{"toolbox/authInvoke": []string{"auth1", "auth2"}},
		},
		{
			desc:        "with auth param metadata",
			name:        "basic",
			description: "foo bar",
			authInvoke:  []string{},
			params:      parameters.Parameters{parameters.NewStringParameterWithAuth("string-param", "string parameter", authServices)},
			annotations: nil,
			wantMetadata: map[string]any{
				"toolbox/authParam": map[string][]string{
					"string-param": {"my-google-auth-service", "other-auth-service"},
				},
			},
		},
		{
			desc:        "with auth invoke and auth param metadata",
			name:        "basic",
			description: "foo bar",
			authInvoke:  []string{"auth1", "auth2"},
			params:      parameters.Parameters{parameters.NewStringParameterWithAuth("string-param", "string parameter", authServices)},
			annotations: nil,
			wantMetadata: map[string]any{
				"toolbox/authInvoke": []string{"auth1", "auth2"},
				"toolbox/authParam": map[string][]string{
					"string-param": {"my-google-auth-service", "other-auth-service"},
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got := tools.GetMcpManifest(tc.name, tc.description, tc.authInvoke, tc.params, tc.annotations)
			gotM := got.Metadata
			if diff := cmp.Diff(tc.wantMetadata, gotM); diff != "" {
				t.Fatalf("unexpected metadata (-want +got):\n%s", diff)
			}

			if got.Annotations != nil {
				annotations, _ := json.Marshal(got.Annotations)
				if diff := cmp.Diff(tc.wantAnnotations, annotations); diff != "" {
					t.Fatalf("unexpected annotations (-want +got):\n%s", diff)
				}
			}

		})
	}
}

func TestCloneAndFilter(t *testing.T) {
	manifest := tools.McpManifest{
		Name:        "test-tool",
		Description: "test tool description",
		InputSchema: parameters.McpToolsSchema{
			Type: "object",
			Properties: map[string]parameters.ParameterMcpManifest{
				"param1": {Type: "string", Description: "param1 desc"},
				"param2": {Type: "string", Description: "param2 desc"},
				"param3": {Type: "string", Description: "param3 desc"},
			},
			Required: []string{"param1", "param2"},
		},
	}

	params := map[string]string{
		"param2": "value2",
		"param4": "value4",
	}

	cloned := manifest.CloneAndFilter(params)

	// Verify properties removed
	if _, exists := cloned.InputSchema.Properties["param2"]; exists {
		t.Errorf("param2 should have been removed from properties")
	}
	if _, exists := cloned.InputSchema.Properties["param1"]; !exists {
		t.Errorf("param1 should not have been removed from properties")
	}
	if _, exists := cloned.InputSchema.Properties["param3"]; !exists {
		t.Errorf("param3 should not have been removed from properties")
	}

	// Verify required removed
	expectedRequired := []string{"param1"}
	if diff := cmp.Diff(expectedRequired, cloned.InputSchema.Required); diff != "" {
		t.Errorf("unexpected required list (-want +got):\n%s", diff)
	}

	// Verify original not modified
	if _, exists := manifest.InputSchema.Properties["param2"]; !exists {
		t.Errorf("original manifest should not have been modified (param2 missing)")
	}
	if len(manifest.InputSchema.Required) != 2 {
		t.Errorf("original manifest required list should not have been modified")
	}
}
