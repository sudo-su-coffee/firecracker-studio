# Firecracker Studio Update Strategy

## Windows desktop application

Firecracker Studio should use GitHub Releases as the public distribution channel. Every release must contain a versioned NSIS installer, a standalone executable, and a SHA256 checksum file. The application can check a small release metadata endpoint on startup or from **Settings → Updates**, compare the current semantic version with the latest stable release, and show a non-blocking update banner.

The update must be user-approved. The user can download the installer, verify its checksum, close Firecracker Studio, and run the installer. NSIS can install the new version over the existing installation while preserving the user’s server profiles and desktop preferences. The application should never silently replace its own executable or execute an unverified download.

## Recommended update flow

| Stage | Behavior |
|---|---|
| Check | Query the latest stable GitHub Release metadata on startup or manually |
| Notify | Show the available version, release notes, and download size |
| Approve | User chooses **Download update** or ignores it |
| Verify | Check HTTPS download and SHA256 against the release checksum file |
| Install | Close the app and launch the signed NSIS installer |
| Recover | Keep the previous installation available until the new installer succeeds |

## Signing

Before broad distribution, Windows installers should be Authenticode-signed with a certificate owned by BlackLoverTech. Signing reduces SmartScreen friction and lets users verify publisher identity. Signing credentials must remain in GitHub Actions secrets and must never be committed to the repository.

## Worker updates

The desktop application and its local Firecracker runtime are separate update concerns. Updating the Windows UI must not silently replace the user’s WSL2 runtime, Firecracker binary, jailer, kernel, rootfs, volumes, or running microVMs. A runtime update should be an explicit operation that displays the target version, compatibility requirements, restart impact, and rollback plan.

For remote Firecracker workers, the worker can expose a version and compatibility range. Studio should warn when the UI and worker versions are incompatible, but it should not force an update. For the local WSL2 runtime, Studio should provide a user-approved, signed, transactional runtime update that preserves images, volumes, snapshots, and running-workload safety.

## Release channels

The stable channel should use tags such as `v1.0.1`. A preview channel can use prereleases such as `v1.1.0-rc.1`, but preview releases must never replace the stable update path. Release notes should clearly distinguish desktop changes, worker changes, Firecracker compatibility changes, and known host limitations.

## Current release

Windows v1.0.1 is published at https://github.com/sudo-su-coffee/firecracker-studio/releases/tag/v1.0.1. The current release includes the manual remote-worker connection flow, local runtime installer, and first-run onboarding panel. It does not yet contain an in-app update checker or Authenticode signing.
