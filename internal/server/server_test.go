// SPDX-FileCopyrightText: 2025 Stanislav Zaprudskiy <stanislav.zaprudskiy@gmail.com>
//
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/applicationcredentials"
	"github.com/stanislav-zaprudskiy/secrets-store-csi-driver-provider-openstack/internal/provider"
	"sigs.k8s.io/secrets-store-csi-driver/provider/v1alpha1"
)

var (
	cmpJSONOpts = cmp.Options{
		cmp.FilterValues(func(x, y []byte) bool {
			return json.Valid(x) && json.Valid(y)
		}, cmp.Transformer("ParseJSON", func(in []byte) (out interface{}) {
			if err := json.Unmarshal(in, &out); err != nil {
				panic(err) // should never occur given previous filter to ensure valid JSON
			}
			return out
		})),
	}
)

type MockedProviderClient struct {
	MockedCreateApplicationCredential func(ctx context.Context, auth map[string]string, createOpts applicationcredentials.CreateOptsBuilder) (*applicationcredentials.ApplicationCredential, error)
}

func (m MockedProviderClient) CreateApplicationCredential(ctx context.Context, auth map[string]string, createOpts applicationcredentials.CreateOptsBuilder) (*applicationcredentials.ApplicationCredential, error) {
	return m.MockedCreateApplicationCredential(ctx, auth, createOpts)
}

func TestVersion(t *testing.T) {
	server := NewServer(provider.Client{})
	version, err := server.Version(context.TODO(), &v1alpha1.VersionRequest{})
	if err != nil {
		t.Fatal(err)
	}

	if version == nil {
		t.Fatal("version should not be nil")
	}

	if version.Version != "v1alpha1" {
		t.Fatal("version.Version should be v1alpha1")
	}

	if version.RuntimeVersion == "" {
		t.Fatal("version.RuntimeVersion should not be empty (and must be semver-compatible)")
	}

	if version.RuntimeName != "secrets-store-csi-driver-provider-openstack" {
		t.Fatal("version.RuntimeName should be secrets-store-csi-driver-provider-openstack")
	}
}

func TestMount(t *testing.T) {
	mountRequest := &v1alpha1.MountRequest{
		Attributes: func() string {
			attributes := map[string]string{
				"applicationCredentials": `- fileName: secure-clouds.yaml
`,
			}
			data, _ := json.Marshal(attributes)
			return string(data)
		}(),
		// TODO: consume secrets
		Secrets:    "{}",
		TargetPath: "/openstack-auth",
		Permission: "640",
		// CurrentObjectVersion: []*v1alpha1.ObjectVersion{},
	}
	wantMountResponse := &v1alpha1.MountResponse{
		ObjectVersion: []*v1alpha1.ObjectVersion{
			{
				// TODO: populate ObjectVersion
				Id:      "",
				Version: "",
			},
		},
		Files: []*v1alpha1.File{
			{
				Path: "secure-clouds.yaml",
				// TODO: validate Contents
				Contents: []byte(`qwe`),
			},
		},
	}

	server := NewServer(MockedProviderClient{
		MockedCreateApplicationCredential: func(ctx context.Context, auth map[string]string, createOpts applicationcredentials.CreateOptsBuilder) (*applicationcredentials.ApplicationCredential, error) {
			return &applicationcredentials.ApplicationCredential{}, nil
		},
	})
	gotMountResponse, _ := server.Mount(context.TODO(), mountRequest)

	wantJSON, _ := json.Marshal(wantMountResponse)
	gotJSON, _ := json.Marshal(gotMountResponse)

	if diff := cmp.Diff(wantJSON, gotJSON, cmpJSONOpts); diff != "" {
		t.Errorf("Mount() mismatch (-want, +got):\n%s", diff)
	}
}
