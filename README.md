# InfiniBand device plugin for Kubernetes

## Disclaimer

This plugin is created for a very specific machine learning case and is not intended for the generic usage. Please read the README carefully before using it.

## Introduction

For now, Kubernetes [does not support](https://github.com/kubernetes/kubernetes/issues/5607) binding host devices to containers.

In the case of distributed deep learning trainings, one should pass the InfiniBand devices to the container with training in order to use it. For now, this is only possible by using privileged containers, which may be too risky for someone.

This plugin solves the problem by introducing a new [device plugin](https://kubernetes.io/docs/concepts/extend-kubernetes/compute-storage-net/device-plugins/) for Kubernetes which is capable of binding InfiniBand devices to containers. When the plugin is installed, the new resource `ib.plugin/infiniband` appears in the system. If the container requests this resource, the plugin will bind the InfiniBand devices to the container.

Note, that if resource is requested by the container, the plugin will bind **all** InfiniBand devices to the container. This is fine for the case of modern distributed trainings because usually all the GPU and InfiniBand devices are used by the training. The fine-grained selection of devices is not supported intentionally. For the machine learning use cases, the set of the passed InfiniBand devices should be the same as the set of the selected GPUs. It seems hard to implement such a logic in Kubernetes and it does not seem to be necessary for now.

Also note that if you want to use InfiniBand, [you will likely need](https://catalog.ngc.nvidia.com/orgs/hpc/containers/preflightcheck) `IPC_LOCK` capability. It is not set by the plugin, so you should set it manually in the pod security context.

## How to install

TODO: Push the nightly build to some public registry.

For now, you need to build the plugin manually. This can be done in two commands.

```bash
CGO_ENABLED=0 go build -o k8s-ib-device-plugin
docker build .
```

After pushing the resulting image to the registry, replace the image in the `k8s-ib-device-plugin.yaml` file and apply it to the cluster.

```bash
kubectl apply -f k8s-ib-device-plugin.yaml
```

You should see the new resource in the system.

## Options

The plugin has the following options:
* `-resource-name` (default: `ib.plugin/infiniband`) - the name of the resource that the plugin will provide.
* `-resouce-amount` (default: `1`) - the amount of the resource that the plugin will provide. You will typically need either `1` (in this case only one pod using InfiniBand will be scheduled to the node and will use all the devices) or some big number (in this case multiple pods using InfiniBand will be scheduled to the node and will share the devices).

## License

[MIT](LICENSE)
