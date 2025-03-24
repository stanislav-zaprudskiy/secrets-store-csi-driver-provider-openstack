// SPDX-FileCopyrightText: 2025 Stanislav Zaprudskiy <stanislav.zaprudskiy@gmail.com>
//
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"

	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/applicationcredentials"
)

type ProviderClient interface {
	CreateApplicationCredential(ctx context.Context, auth map[string]string, createOpts applicationcredentials.CreateOptsBuilder) (*applicationcredentials.ApplicationCredential, error)
}

type Client struct{}

func (c Client) CreateApplicationCredential(ctx context.Context, auth map[string]string, createOpts applicationcredentials.CreateOptsBuilder) (*applicationcredentials.ApplicationCredential, error) {
	// TODO: real implementation
	return nil, nil
}
