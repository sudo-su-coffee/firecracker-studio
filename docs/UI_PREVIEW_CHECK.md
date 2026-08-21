# Firecracker Studio v2.0.1 UI Preview Check

**Date:** 2026-08-19

The v2.0.1 browser UI is served by the embedded Go web server. It does not require Wails, a native desktop shell, or direct browser access to Firecracker sockets.

The primary navigation is intentionally small: **Dashboard, Images, Workloads, and Convert**. The dashboard shows the server connection, runtime readiness, host metrics, recent workloads, and recent lifecycle events. Images shows the local artifact catalog and runtime asset metadata. Workloads provides create, start, stop, inspect, and delete actions. Convert provides the supported OCI-to-artifact workflow.

The runtime installation remains separate from the Studio server. When Firecracker, jailer, KVM, kernel, or rootfs assets are unavailable, the readiness response reports the missing prerequisite instead of presenting a false ready state.

The workload detail panel labels existing output as **Studio lifecycle events**. It does not claim that lifecycle events are guest application stdout. Networking, guest logs, snapshots, remote workers, and other advanced capabilities must only be shown as available when the installed runtime and host configuration support them.

The v2.0.1 UI is intentionally a local control surface rather than a cloud dashboard. Porter, multi-node orchestration, accounts, billing, Marketplace dependency, and other SaaS concerns are outside this release.
