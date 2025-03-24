// SPDX-FileCopyrightText: 2025 Stanislav Zaprudskiy <stanislav.zaprudskiy@gmail.com>
//
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/applicationcredentials"
	"github.com/stanislav-zaprudskiy/secrets-store-csi-driver-provider-openstack/internal/provider"
	"sigs.k8s.io/secrets-store-csi-driver/provider/v1alpha1"
	"sigs.k8s.io/yaml"
)

type CSIDriverProviderServer struct {
	v1alpha1.UnimplementedCSIDriverProviderServer
	ProviderClient provider.ProviderClient
}

func NewServer(providerClient provider.ProviderClient) *CSIDriverProviderServer {
	return &CSIDriverProviderServer{
		ProviderClient: providerClient,
	}
}

func (s *CSIDriverProviderServer) Version(ctx context.Context, req *v1alpha1.VersionRequest) (*v1alpha1.VersionResponse, error) {
	return &v1alpha1.VersionResponse{
		Version:        "v1alpha1",
		RuntimeName:    "secrets-store-csi-driver-provider-openstack",
		RuntimeVersion: "0.0.1",
	}, nil
}

func (s *CSIDriverProviderServer) Mount(ctx context.Context, req *v1alpha1.MountRequest) (*v1alpha1.MountResponse, error) {
	var mountDir string
	var attributes, secrets map[string]string
	var filePermission os.FileMode
	var err error

	if mountDir = req.GetTargetPath(); mountDir == "" {
		return nil, fmt.Errorf("request should have a target mount path")
	}

	// attributes correspond to SecretProviderClass.spec.attributes
	if err = json.Unmarshal([]byte(req.GetAttributes()), &attributes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal attributes, error: %w", err)
	}

	// secrets is the Secret content referenced in nodePublishSecretRef Secret data
	if err = json.Unmarshal([]byte(req.GetSecrets()), &secrets); err != nil {
		return nil, fmt.Errorf("failed to unmarshal nodePublishSecretRef secrets, error: %w", err)
	}
	if secrets == nil {
		return nil, fmt.Errorf("secrets should be provided via volume.csi.nodePublishSecretRef.name")
	}

	if err = json.Unmarshal([]byte(req.GetPermission()), &filePermission); err != nil {
		return nil, fmt.Errorf("failed to unmarshal file permission, error: %w", err)
	}

	// req.GetCurrentObjectVersion()

	applicationCredentialAttribute, ok := attributes["applicationCredentials"]
	// if Barbican `secrets` are added, it would have to account for both
	if !ok || applicationCredentialAttribute == "" {
		return nil, fmt.Errorf("applicationCredentials should be provided via SecretProviderClass.spec.attributes.applicationCredentials")
	}
	var applicationCredentialsObjects []*ApplicationCredentialObject
	err = yaml.Unmarshal([]byte(applicationCredentialAttribute), &applicationCredentialsObjects)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal applicationCredentials, error: %w", err)
	}

	mountResponse := &v1alpha1.MountResponse{}

	for _, applicationCredentialObject := range applicationCredentialsObjects {
		applicationCredential, err := s.ProviderClient.CreateApplicationCredential(ctx, secrets, applicationCredentialObject)
		if err != nil {
			return nil, fmt.Errorf("failed to create application credential %+v, error: %w", applicationCredentialObject, err)
		}

		contents, err := renderContents(applicationCredentialObject, applicationCredential)
		if err != nil {
			return nil, fmt.Errorf("failed to render contents for application credential %+v, error: %w", applicationCredentialObject, err)
		}

		file := &v1alpha1.File{
			Contents: []byte(contents),
			Path:     applicationCredentialObject.FileName,
		}
		mountResponse.Files = append(mountResponse.Files, file)

		objectVersion := &v1alpha1.ObjectVersion{
			Id:      "",
			Version: "",
		}
		mountResponse.ObjectVersion = append(mountResponse.ObjectVersion, objectVersion)
	}

	return mountResponse, nil
}

func renderContents(applicationCredentialObject *ApplicationCredentialObject, applicationCredential *applicationcredentials.ApplicationCredential) (string, error) {
	// TODO: real implementation
	return "qwe", nil
}
