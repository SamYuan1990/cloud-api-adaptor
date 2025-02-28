# Setup instructions

- KVM host with libvirt configured.
- Libvirt network and storage pool created
- A base storage volume created for POD VM

## Creation of base storage volume

- Ubuntu 20.04 VM with minimum 50GB disk and the following packages installed
  - `cloud-image-utils`
  - `qemu-system-x86`

- Install packer on the VM by following the instructions in the following [link](https://learn.hashicorp.com/tutorials/packer/get-started-install-cli)

- Create qcow2 image by executing the following command
```
cd image
CLOUD_PROVIDER=libvirt make build
```

The image will be available under the `output` directory

- Copy the qcow2 image to the libvirt machine

- Create volume
```
export IMAGE=<full-path-to-qcow2>

virsh vol-create-as --pool default --name podvm-base.qcow2 --capacity 107374182400 --allocation 2361393152 --prealloc-metadata --format qcow2
virsh vol-upload --vol podvm-base.qcow2 $IMAGE --pool default --sparse
```

If you want to set default password for debugging then you can use guestfish to edit the qcow2 and make any suitable changes.

# Running cloud-api-adaptor

Export the required environment variables.

```
export LIBVIRT_URI=REPLACE_ME
export LIBVIRT_POOL=REPLACE_ME
export LIBVIRT_NET=REPLACE_ME
```
Note that the `LIBVIRT_URI` should be of the form - `qemu+ssh://root@<LIBVIRT_HOST_ADDR>/system`.

Run the binary.

```
mkdir -p /opt/data-dir

./cloud-api-adaptor libvirt \
    -uri ${LIBVIRT_URI}  \
    -data-dir /opt/data-dir \
    -pods-dir /run/peerpod/pods \
    -network-name ${LIBVIRT_NET} \
    -pool-name ${LIBVIRT_POOL} \
    -socket /run/peerpod/hypervisor.sock

```

