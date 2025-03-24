// SPDX-FileCopyrightText: 2025 Stanislav Zaprudskiy <stanislav.zaprudskiy@gmail.com>
//
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/applicationcredentials"
)

type ApplicationCredentialObject struct {
	FileName string `json:"fileName" yaml:"fileName"`
	// Template
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
