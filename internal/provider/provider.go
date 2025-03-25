// SPDX-FileCopyrightText: 2025 Stanislav Zaprudskiy <stanislav.zaprudskiy@gmail.com>
//
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"errors"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/applicationcredentials"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
)

type ProviderClient interface {
	CreateApplicationCredential(ctx context.Context, auth map[string]string, createOpts applicationcredentials.CreateOptsBuilder) (*applicationcredentials.ApplicationCredential, *gophercloud.ServiceClient, error)
}

type Client struct{}

func (c Client) CreateApplicationCredential(ctx context.Context, auth map[string]string, createOpts applicationcredentials.CreateOptsBuilder) (*applicationcredentials.ApplicationCredential, *gophercloud.ServiceClient, error) {
	providerClient, identityClient, err := newGophercloudClients(ctx, auth)
	if err != nil {
		return nil, identityClient, err
	}

	currentToken, ok := providerClient.GetAuthResult().(tokens.CreateResult)
	if !ok {
		return nil, identityClient, errors.New("failed to get auth result of current token")
	}

	currentUser, err := currentToken.ExtractUser()
	if err != nil {
		return nil, identityClient, err
	}

	applicationCredential, err := applicationcredentials.Create(ctx, identityClient, currentUser.ID, createOpts).Extract()

	return applicationCredential, identityClient, err
}

func newGophercloudClients(ctx context.Context, auth map[string]string) (*gophercloud.ProviderClient, *gophercloud.ServiceClient, error) {
	authOptions, err := AuthOptionsFromMap(auth)
	if err != nil {
		return nil, nil, err
	}

	providerClient, err := openstack.AuthenticatedClient(ctx, authOptions)
	if err != nil {
		return providerClient, nil, err
	}

	eo := gophercloud.EndpointOpts{
		Availability: gophercloud.Availability(auth["OS_INTERFACE"]),
		Region:       auth["OS_REGION_NAME"],
	}

	identityClient, err := openstack.NewIdentityV3(providerClient, eo)
	if err != nil {
		return providerClient, identityClient, err
	}
	return providerClient, identityClient, nil
}

// AuthOptionsFromMap is a copy of openstack.AuthOptionsFromEnv, returning
// gophercloud.AuthOptions for input map which items represent corresponding OS
// environment variable names and values
// https://github.com/gophercloud/gophercloud/blob/9e4535f6f0974f25793e62c03e2794097dfa1a43/openstack/auth_env.go#L36
// ¯\_(ツ)_/¯
func AuthOptionsFromMap(authMap map[string]string) (gophercloud.AuthOptions, error) {
	authURL := authMap["OS_AUTH_URL"]
	username := authMap["OS_USERNAME"]
	userID := authMap["OS_USERID"]
	password := authMap["OS_PASSWORD"]
	passcode := authMap["OS_PASSCODE"]
	tenantID := authMap["OS_TENANT_ID"]
	tenantName := authMap["OS_TENANT_NAME"]
	domainID := authMap["OS_DOMAIN_ID"]
	domainName := authMap["OS_DOMAIN_NAME"]
	applicationCredentialID := authMap["OS_APPLICATION_CREDENTIAL_ID"]
	applicationCredentialName := authMap["OS_APPLICATION_CREDENTIAL_NAME"]
	applicationCredentialSecret := authMap["OS_APPLICATION_CREDENTIAL_SECRET"]
	systemScope := authMap["OS_SYSTEM_SCOPE"]

	// If OS_PROJECT_ID is set, overwrite tenantID with the value.
	if v := authMap["OS_PROJECT_ID"]; v != "" {
		tenantID = v
	}

	// If OS_PROJECT_NAME is set, overwrite tenantName with the value.
	if v := authMap["OS_PROJECT_NAME"]; v != "" {
		tenantName = v
	}

	if authURL == "" {
		err := gophercloud.ErrMissingEnvironmentVariable{
			EnvironmentVariable: "OS_AUTH_URL",
		}
		return gophercloud.AuthOptions{}, err
	}

	if userID == "" && username == "" {
		// Empty username and userID could be ignored, when applicationCredentialID and applicationCredentialSecret are set
		if applicationCredentialID == "" && applicationCredentialSecret == "" {
			err := gophercloud.ErrMissingAnyoneOfEnvironmentVariables{
				EnvironmentVariables: []string{"OS_USERID", "OS_USERNAME"},
			}
			return gophercloud.AuthOptions{}, err
		}
	}

	if password == "" && passcode == "" && applicationCredentialID == "" && applicationCredentialName == "" {
		err := gophercloud.ErrMissingEnvironmentVariable{
			// silently ignore TOTP passcode warning, since it is not a common auth method
			EnvironmentVariable: "OS_PASSWORD",
		}
		return gophercloud.AuthOptions{}, err
	}

	if (applicationCredentialID != "" || applicationCredentialName != "") && applicationCredentialSecret == "" {
		err := gophercloud.ErrMissingEnvironmentVariable{
			EnvironmentVariable: "OS_APPLICATION_CREDENTIAL_SECRET",
		}
		return gophercloud.AuthOptions{}, err
	}

	if domainID == "" && domainName == "" && tenantID == "" && tenantName != "" {
		err := gophercloud.ErrMissingEnvironmentVariable{
			EnvironmentVariable: "OS_PROJECT_ID",
		}
		return gophercloud.AuthOptions{}, err
	}

	if applicationCredentialID == "" && applicationCredentialName != "" && applicationCredentialSecret != "" {
		if userID == "" && username == "" {
			return gophercloud.AuthOptions{}, gophercloud.ErrMissingAnyoneOfEnvironmentVariables{
				EnvironmentVariables: []string{"OS_USERID", "OS_USERNAME"},
			}
		}
		if username != "" && domainID == "" && domainName == "" {
			return gophercloud.AuthOptions{}, gophercloud.ErrMissingAnyoneOfEnvironmentVariables{
				EnvironmentVariables: []string{"OS_DOMAIN_ID", "OS_DOMAIN_NAME"},
			}
		}
	}

	var scope *gophercloud.AuthScope
	if systemScope == "all" {
		scope = &gophercloud.AuthScope{
			System: true,
		}
	}

	ao := gophercloud.AuthOptions{
		IdentityEndpoint:            authURL,
		UserID:                      userID,
		Username:                    username,
		Password:                    password,
		Passcode:                    passcode,
		TenantID:                    tenantID,
		TenantName:                  tenantName,
		DomainID:                    domainID,
		DomainName:                  domainName,
		ApplicationCredentialID:     applicationCredentialID,
		ApplicationCredentialName:   applicationCredentialName,
		ApplicationCredentialSecret: applicationCredentialSecret,
		Scope:                       scope,
	}

	return ao, nil
}
