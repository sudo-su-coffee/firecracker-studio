# Real Firecracker VM smoke test

The current v1.2.x worker allocates a Unix socket and sends API requests, but it does not yet launch a Firecracker process or configure the kernel/rootfs automatically. Therefore use the two flows below: direct Firecracker boot validates the real microVM, while the Studio API flow validates that a workload appears in the Web UI.

## 1. Start Firecracker Studio

```bash
sudo systemctl restart firecracker-studio
curl -fsS http://127.0.0.1:7822/api/v1/health
```

Open `http://127.0.0.1:7822` and select **Workloads**.

## 2. Find or download boot assets

```bash
mkdir -p "$HOME/firecracker-images"
cd "$HOME/firecracker-images"
wget -O vmlinux https://s3.amazonaws.com/spec.ccfc.min/img/hello/kernel/hello-vmlinux.bin
wget -O rootfs.ext4 https://s3.amazonaws.com/spec.ccfc.min/img/hello/fsfiles/hello-rootfs.ext4
chmod 0644 vmlinux rootfs.ext4
ls -lh vmlinux rootfs.ext4
```

## 3. Boot a real Firecracker microVM directly

Run this in one terminal:

```bash
export API_SOCKET=/tmp/firecracker-studio-real-test.sock
rm -f "$API_SOCKET"
firecracker --api-sock "$API_SOCKET"
```

Run this in a second terminal:

```bash
API_SOCKET=/tmp/firecracker-studio-real-test.sock
fc() { curl --silent --show-error --fail --unix-socket "$API_SOCKET" -X PUT "http://localhost$1" -H 'Content-Type: application/json' --data "$2"; echo; }

fc /machine-config '{"vcpu_count":1,"mem_size_mib":512,"smt":false}'
fc /boot-source "{\"kernel_image_path\":\"$HOME/firecracker-images/vmlinux\",\"boot_args\":\"console=ttyS0 reboot=k panic=1 pci=off\"}"
fc /drives/rootfs "{\"drive_id\":\"rootfs\",\"path_on_host\":\"$HOME/firecracker-images/rootfs.ext4\",\"is_root_device\":true,\"is_read_only\":false}"
fc /actions '{"action_type":"InstanceStart"}'
```

The Firecracker terminal should show that the API received the machine configuration, boot source, rootfs drive, and `InstanceStart` action.

## 4. Make a Studio workload appear in the Web UI

This creates a visible API-managed workload. In the current v1.2.x release it is a UI/control-plane smoke test, not the same process as the direct Firecracker VM above:

```bash
cd /path/to/firecracker-studio
STUDIO_URL=http://127.0.0.1:7822 ./scripts/test-vm-workflows.sh empty
```

Or create a workload with the real rootfs path as a visible artifact label:

```bash
ROOTFS="$HOME/firecracker-images/rootfs.ext4"
curl -fsS -X POST http://127.0.0.1:7822/api/v1/vms \
  -H 'Content-Type: application/json' \
  --data "{\"artifactDigest\":\"file://$ROOTFS\",\"imageReference\":\"firecracker-hello-real-test\",\"vcpus\":1,\"memoryMiB\":512}" | python3 -m json.tool
```

Return to **Workloads** in the Web UI. The workload should show `firecracker-hello-real-test`, its created state, ports if configured, and lifecycle logs.

Do not click **Start** on this API-created record yet; the current worker does not automatically attach the kernel/rootfs or launch the Firecracker process. The direct Firecracker process from step 3 is the real running VM.

## 5. Stop the direct test VM

In the second terminal:

```bash
API_SOCKET=/tmp/firecracker-studio-real-test.sock
curl --silent --show-error --unix-socket "$API_SOCKET" -X PUT http://localhost/actions \
  -H 'Content-Type: application/json' \
  --data '{"action_type":"SendCtrlAltDel"}'
```

Then stop the first terminal with `Ctrl+C`.
