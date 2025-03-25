// SPDX-FileCopyrightText: 2025 Stanislav Zaprudskiy <stanislav.zaprudskiy@gmail.com>
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/stanislav-zaprudskiy/secrets-store-csi-driver-provider-openstack/internal/provider"
	"github.com/stanislav-zaprudskiy/secrets-store-csi-driver-provider-openstack/internal/server"
	"google.golang.org/grpc"

	"sigs.k8s.io/secrets-store-csi-driver/provider/v1alpha1"
)

var (
	// might be reasonable to migrate /var/run/secrets-store-csi-provider path,
	// https://github.com/kubernetes-sigs/secrets-store-csi-driver/issues/823
	volumePath = flag.String("volume-path", "/etc/kubernetes/secrets-store-csi-providers", "path to directory where to serve the provider socket")
)

func main() {
	flag.Parse()

	endpoint := fmt.Sprintf("%s/openstack.sock", *volumePath)
	_ = os.Remove(endpoint)
	grpcSrv := grpc.NewServer()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT, os.Interrupt)
	go func() {
		sig := <-sigs
		slog.Info("Received signal to terminate", "signal", sig)
		grpcSrv.GracefulStop()
	}()

	listener, err := net.Listen("unix", endpoint)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	defer func() {
		listener.Close()
		os.Remove(endpoint)
	}()
	slog.Info("Listening for connections", "address", listener.Addr())

	providerServer := server.NewServer(provider.Client{})
	v1alpha1.RegisterCSIDriverProviderServer(grpcSrv, providerServer)

	if err := grpcSrv.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
