# Terra OS

![terra](iso/splash.png)

Modern, minimal operating system (we've heard that before) optimized for containers within the Stellar Project.

## Kernel

We use the latest stable kernel with a stripped down config.
The config is optimized for the KSPP guidelines.

### Kernel Patches

* wireguard

## OS

### Services

* containerd - runtime
	* orbit
	* stellar store
	* stellar dns
* buildkit - image builder
* Prometheus Node Exporter - node metrics

### Binaries

* criu - checkpoint and restore
* cni plugins - networking
* vab - image build frontend
