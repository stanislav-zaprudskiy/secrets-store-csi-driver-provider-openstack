// SPDX-FileCopyrightText: 2025 Stanislav Zaprudskiy <stanislav.zaprudskiy@gmail.com>
//
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gophercloud/gophercloud/v2"

	"github.com/google/go-cmp/cmp"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/applicationcredentials"
	"github.com/stanislav-zaprudskiy/secrets-store-csi-driver-provider-openstack/internal/provider"
	"google.golang.org/protobuf/testing/protocmp"
	"sigs.k8s.io/secrets-store-csi-driver/provider/v1alpha1"
)

type MockedProviderClient struct {
	MockedCreateApplicationCredential func(ctx context.Context, auth map[string]string, createOpts applicationcredentials.CreateOptsBuilder) (*applicationcredentials.ApplicationCredential, *gophercloud.ServiceClient, error)
}

func (m MockedProviderClient) CreateApplicationCredential(ctx context.Context, auth map[string]string, createOpts applicationcredentials.CreateOptsBuilder) (*applicationcredentials.ApplicationCredential, *gophercloud.ServiceClient, error) {
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
	tests := map[string]struct {
		applicationCredentials string
		filePath               string
		contents               string
		server                 *CSIDriverProviderServer
	}{
		"applicationCredential with template": {
			applicationCredentials: `
- fileName: secure-clouds.yaml
  template: "qwe"
`,
			filePath: "secure-clouds.yaml",
			contents: "qwe",
			server: NewServer(MockedProviderClient{
				MockedCreateApplicationCredential: func(ctx context.Context, auth map[string]string, createOpts applicationcredentials.CreateOptsBuilder) (*applicationcredentials.ApplicationCredential, *gophercloud.ServiceClient, error) {
					return &applicationcredentials.ApplicationCredential{}, &gophercloud.ServiceClient{}, nil
				},
			}),
		},

		"applicationCredential with default template": {
			applicationCredentials: `- fileName: secure-clouds.yaml
`,
			filePath: "secure-clouds.yaml",
			contents: `clouds:
  secrets-store-csi:
    auth:
      application_credential_id: ""
      application_credential_secret: ""
      auth_url: ""
    auth_type: "v3applicationcredential"
`,
			server: NewServer(MockedProviderClient{
				MockedCreateApplicationCredential: func(ctx context.Context, auth map[string]string, createOpts applicationcredentials.CreateOptsBuilder) (*applicationcredentials.ApplicationCredential, *gophercloud.ServiceClient, error) {
					return &applicationcredentials.ApplicationCredential{}, &gophercloud.ServiceClient{}, nil
				},
			}),
		},

		"applicationCredential with default template and non zero AC": {
			applicationCredentials: `
- fileName: secure-clouds.yaml
`,
			filePath: "secure-clouds.yaml",
			contents: `clouds:
  secrets-store-csi:
    auth:
      application_credential_id: "abcdef1234"
      application_credential_secret: "random-generated-secret"
      auth_url: "http://localhost:5000/v3/"
    auth_type: "v3applicationcredential"
`,
			server: NewServer(MockedProviderClient{
				MockedCreateApplicationCredential: func(ctx context.Context, auth map[string]string, createOpts applicationcredentials.CreateOptsBuilder) (*applicationcredentials.ApplicationCredential, *gophercloud.ServiceClient, error) {
					ac := &applicationcredentials.ApplicationCredential{
						ID:     "abcdef1234",
						Secret: "random-generated-secret",
					}
					sc := &gophercloud.ServiceClient{
						ResourceBase: "http://localhost:5000/v3/",
					}
					return ac, sc, nil
				},
			}),
		},

		"applicationCredential with custom template and non zero AC": {
			applicationCredentials: `
- fileName: keystone.conf
  template: |
    [keystone_authtoken]
    auth_url = {{ .AuthInfo.AuthURL }}
    auth_type = {{ .AuthType }}
    application_credential_id = {{ .AuthInfo.ApplicationCredentialID }}
    application_credential_secret= {{ .AuthInfo.ApplicationCredentialSecret }}
`,
			filePath: "keystone.conf",
			contents: `[keystone_authtoken]
auth_url = https://keystone.server/identity/v3/
auth_type = v3applicationcredential
application_credential_id = 6cb5fa6a13184e6fab65ba2108adf50c
application_credential_secret= glance_secret
`,
			server: NewServer(MockedProviderClient{
				MockedCreateApplicationCredential: func(ctx context.Context, auth map[string]string, createOpts applicationcredentials.CreateOptsBuilder) (*applicationcredentials.ApplicationCredential, *gophercloud.ServiceClient, error) {
					ac := &applicationcredentials.ApplicationCredential{
						ID:     "6cb5fa6a13184e6fab65ba2108adf50c",
						Secret: "glance_secret",
					}
					sc := &gophercloud.ServiceClient{
						ResourceBase: "https://keystone.server/identity/v3/",
					}
					return ac, sc, nil
				},
			}),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			mountRequest := &v1alpha1.MountRequest{
				Attributes: func() string {
					attributes := map[string]string{
						"applicationCredentials": test.applicationCredentials,
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
						Path:     test.filePath,
						Contents: []byte(test.contents),
					},
				},
			}

			gotMountResponse, err := test.server.Mount(context.TODO(), mountRequest)
			if err != nil {
				t.Fatalf("MountRequest failed: %v", err)
			}
			if diff := cmp.Diff(wantMountResponse, gotMountResponse, protocmp.Transform()); diff != "" {
				t.Errorf("Mount() mismatch (-want, +got):\n%s", diff)
			}
		})
	}
}
