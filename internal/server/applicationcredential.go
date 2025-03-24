// SPDX-FileCopyrightText: 2025 Stanislav Zaprudskiy <stanislav.zaprudskiy@gmail.com>
//
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"bytes"
	"fmt"
	"math/rand"
	"strconv"
	"text/template"
	"time"

	"github.com/gophercloud/gophercloud/v2"

	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/applicationcredentials"
)

const (
	DefaultTemplate string = `clouds:
{{- range $cloudName, $conf := .Clouds }}
  {{ $cloudName }}:
    auth:
      application_credential_id: "{{ $conf.AuthInfo.ApplicationCredentialID }}"
      application_credential_secret: "{{ $conf.AuthInfo.ApplicationCredentialSecret }}"
      auth_url: "{{ $conf.AuthInfo.AuthURL }}"
    auth_type: "{{ $conf.AuthType }}"
{{- end }}
`
)

type ApplicationCredentialObject struct {
	FileName string  `json:"fileName" yaml:"fileName"`
	Template *string `json:"template,omitempty" yaml:"template,omitempty"`
	// embed ApplicationCredential
}

func (o ApplicationCredentialObject) ToApplicationCredentialCreateMap() (map[string]any, error) {
	// TODO: real implementation
	d := time.Hour * 1
	expiresAt := time.Now().Add(d).Truncate(time.Millisecond).UTC()
	description := "Created with love by secrets-store-csi-driver-provider-openstack"
	createOpts := applicationcredentials.CreateOpts{
		Name:        "secrets-store-csi-" + randomSuffix(5),
		Description: description,
		ExpiresAt:   &expiresAt,
	}

	return createOpts.ToApplicationCredentialCreateMap()
}

func randomSuffix(length int) string {
	c := "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	ts := time.Now().UnixNano()
	r := rand.New(rand.NewSource(ts))
	for i := range b {
		b[i] = c[r.Intn(len(c))]
	}
	return fmt.Sprintf("%s-%s", strconv.FormatInt(ts, 10), string(b))
}

func (o ApplicationCredentialObject) Render(applicationCredential *applicationcredentials.ApplicationCredential, serviceClient *gophercloud.ServiceClient) ([]byte, error) {
	cloudsConfig := newCloudsConfig(applicationCredential, serviceClient)
	return o.executeTemplate(cloudsConfig)
}

func (o ApplicationCredentialObject) executeTemplate(cloudsConfig *Clouds) ([]byte, error) {
	tmpl := DefaultTemplate
	if o.Template != nil {
		tmpl = *o.Template
	}
	var contents []byte

	t, err := template.New("template").Parse(tmpl)
	if err != nil {
		return contents, err
	}

	var buf bytes.Buffer
	err = t.Execute(&buf, cloudsConfig)
	if err != nil {
		return contents, err
	}
	contents = buf.Bytes()

	return contents, nil
}

type Clouds struct {
	Clouds map[string]Cloud
}

type Cloud struct {
	AuthInfo AuthInfo
	AuthType AuthType
}

type AuthType string

const (
	AuthV3ApplicationCredential AuthType = "v3applicationcredential"
)

type AuthInfo struct {
	AuthURL                     string
	ApplicationCredentialID     string
	ApplicationCredentialSecret string
	ApplicationCredentialName   string
}

func newCloudsConfig(applicationCredential *applicationcredentials.ApplicationCredential, identityClient *gophercloud.ServiceClient) *Clouds {
	clouds := Clouds{
		Clouds: make(map[string]Cloud),
	}
	clouds.Clouds["secrets-store-csi"] = Cloud{
		AuthType: AuthV3ApplicationCredential,
		AuthInfo: AuthInfo{
			ApplicationCredentialID:     applicationCredential.ID,
			ApplicationCredentialSecret: applicationCredential.Secret,
			ApplicationCredentialName:   applicationCredential.Name,
			AuthURL:                     identityClient.ResourceBaseURL(),
		},
	}
	return &clouds
}
