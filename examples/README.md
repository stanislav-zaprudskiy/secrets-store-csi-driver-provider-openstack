# README

Create a test cluster first:

```shell
kind create cluster --config examples/kind-config.yaml
```

Deploy the driver and the provider next:

```shell
skaffold dev \
  -m secrets-store-csi-driver \
  -m secrets-store-csi-driver-provider-openstack
```

Then attach a remote debugger, and once connected, deploy the demo app:

```shell
skaffold dev -m demo-app
```

Mounting a socket from a host OS into driver Pods would be a much better
alternative, but it didn't work (macOS, Docker Desktop) for some reason
([this?](https://stackoverflow.com/a/77588828)) - neither from the driver, nor
from e.g. `grpcurl`.

```
Failed to dial target host "unix:/etc/kubernetes/secrets-store-csi-providers/openstack.sock": connection error: desc = "transport: error while dialing: dial unix /etc/kubernetes/secrets-store-csi-providers/openstack.sock: connect: connection refused"

```
