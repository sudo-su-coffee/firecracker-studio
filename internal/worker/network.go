package worker

import (
	"context"
	"crypto/rand"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
)

// NetworkConfig describes the host-side networking Firecracker Studio
// configures for a single microVM: one TAP device bridging the VM's virtio-net
// interface to the host, plus any DNAT port forwards the caller requested.
type NetworkConfig struct {
	TapDevice string
	GuestMac  string
	GuestCIDR string // address assigned to the VM's interface, e.g. "172.16.0.2/30"
	HostCIDR  string // address assigned to the TAP device on the host, e.g. "172.16.0.1/30"
	IfaceID   string // Firecracker network interface id, e.g. "eth0"
}

// NetworkManager creates and tears down TAP devices and DNAT port-forward
// rules for microVMs. It shells out to the standard `ip` and `iptables`
// utilities rather than binding netlink directly, matching the rest of the
// runtime tooling (see internal/runtime/manager.go) and keeping the binary
// free of additional dependencies.
type NetworkManager struct {
	// BridgeCIDRBase is the /24 (or narrower) network Firecracker Studio
	// allocates point-to-point /30s from, one per VM. Defaults to
	// 172.16.0.0/16 if empty.
	BridgeCIDRBase string
}

func NewNetworkManager() *NetworkManager {
	return &NetworkManager{}
}

// Setup creates a TAP device for the given VM, assigns it a host-side IP,
// and returns the configuration to pass into Firecracker's
// /network-interfaces API plus the guest-side boot args needed to configure
// the interface deterministically inside the microVM (no DHCP dependency).
func (n *NetworkManager) Setup(ctx context.Context, vmID string) (NetworkConfig, error) {
	if _, err := exec.LookPath("ip"); err != nil {
		return NetworkConfig{}, fmt.Errorf("networking requires the 'ip' utility (iproute2): %w", err)
	}
	tapName := tapDeviceName(vmID)
	mac, err := randomLocalMAC()
	if err != nil {
		return NetworkConfig{}, fmt.Errorf("generate guest MAC: %w", err)
	}
	hostIP, guestIP, err := allocatePointToPoint(vmID, n.BridgeCIDRBase)
	if err != nil {
		return NetworkConfig{}, fmt.Errorf("allocate VM subnet: %w", err)
	}

	steps := [][]string{
		{"tuntap", "add", "dev", tapName, "mode", "tap"},
		{"addr", "add", hostIP + "/30", "dev", tapName},
		{"link", "set", "dev", tapName, "up"},
	}
	for _, args := range steps {
		if out, err := runIP(ctx, args...); err != nil {
			_ = n.Teardown(context.WithoutCancel(ctx), vmID)
			return NetworkConfig{}, fmt.Errorf("configure TAP device (%s): %s: %w", strings.Join(args, " "), out, err)
		}
	}

	if err := enableIPForwarding(ctx); err != nil {
		_ = n.Teardown(context.WithoutCancel(ctx), vmID)
		return NetworkConfig{}, err
	}
	if err := addMasquerade(ctx, guestIP+"/30"); err != nil {
		_ = n.Teardown(context.WithoutCancel(ctx), vmID)
		return NetworkConfig{}, err
	}

	return NetworkConfig{
		TapDevice: tapName,
		GuestMac:  mac,
		GuestCIDR: guestIP + "/30",
		HostCIDR:  hostIP + "/30",
		IfaceID:   "eth0",
	}, nil
}

// Teardown removes the TAP device and any NAT/DNAT rules associated with the
// VM. It is safe to call even if Setup partially failed or the device was
// never created; individual command failures are ignored so cleanup is
// best-effort and cannot itself block VM deletion.
func (n *NetworkManager) Teardown(ctx context.Context, vmID string) error {
	tapName := tapDeviceName(vmID)
	_, _ = runIP(ctx, "link", "set", "dev", tapName, "down")
	_, _ = runIP(ctx, "tuntap", "del", "dev", tapName, "mode", "tap")
	if _, guestIP, err := allocatePointToPoint(vmID, n.BridgeCIDRBase); err == nil {
		removeMasquerade(ctx, guestIP+"/30")
	}
	return nil
}

// ApplyPortMappings installs DNAT rules on the host forwarding each mapped
// host port to the VM's guest IP/port over the TAP device. Existing rules for
// the VM should be removed first via RemovePortMappings to avoid duplicates
// on VM restart.
func (n *NetworkManager) ApplyPortMappings(ctx context.Context, guestIP string, mappings []PortMapping) error {
	if _, err := exec.LookPath("iptables"); err != nil {
		return fmt.Errorf("port mapping requires the 'iptables' utility: %w", err)
	}
	seen := make(map[string]struct{}, len(mappings))
	for _, m := range mappings {
		proto := m.Protocol
		if proto == "" {
			proto = "tcp"
		}
		key := proto + ":" + strconv.Itoa(m.HostPort)
		if _, exists := seen[key]; exists {
			return fmt.Errorf("duplicate host port mapping %d/%s", m.HostPort, proto)
		}
		seen[key] = struct{}{}
		destination := fmt.Sprintf("%s:%d", guestIP, m.GuestPort)
		for _, chain := range []string{"PREROUTING", "OUTPUT"} {
			args := []string{"-t", "nat", "-A", chain, "-p", proto, "--dport", fmt.Sprintf("%d", m.HostPort), "-j", "DNAT", "--to-destination", destination}
			if out, err := exec.CommandContext(ctx, "iptables", args...).CombinedOutput(); err != nil {
				n.RemovePortMappings(context.WithoutCancel(ctx), guestIP, mappings[:len(seen)])
				return fmt.Errorf("add %s port mapping %d->%d/%s: %s: %w", chain, m.HostPort, m.GuestPort, proto, strings.TrimSpace(string(out)), err)
			}
		}
		forward := []string{"-A", "FORWARD", "-i", "fcs+", "-p", proto, "-d", guestIP, "--dport", fmt.Sprintf("%d", m.GuestPort), "-j", "ACCEPT"}
		if out, err := exec.CommandContext(ctx, "iptables", forward...).CombinedOutput(); err != nil {
			n.RemovePortMappings(context.WithoutCancel(ctx), guestIP, mappings[:len(seen)])
			return fmt.Errorf("allow forwarded port %d->%d/%s: %s: %w", m.HostPort, m.GuestPort, proto, strings.TrimSpace(string(out)), err)
		}
	}
	return nil
}

// RemovePortMappings deletes the DNAT rules previously installed by
// ApplyPortMappings for the same guest IP and mapping set.
func (n *NetworkManager) RemovePortMappings(ctx context.Context, guestIP string, mappings []PortMapping) {
	for _, m := range mappings {
		proto := m.Protocol
		if proto == "" {
			proto = "tcp"
		}
		destination := fmt.Sprintf("%s:%d", guestIP, m.GuestPort)
		for _, chain := range []string{"PREROUTING", "OUTPUT"} {
			args := []string{"-t", "nat", "-D", chain, "-p", proto, "--dport", fmt.Sprintf("%d", m.HostPort), "-j", "DNAT", "--to-destination", destination}
			_, _ = exec.CommandContext(ctx, "iptables", args...).CombinedOutput()
		}
		forward := []string{"-D", "FORWARD", "-i", "fcs+", "-p", proto, "-d", guestIP, "--dport", fmt.Sprintf("%d", m.GuestPort), "-j", "ACCEPT"}
		_, _ = exec.CommandContext(ctx, "iptables", forward...).CombinedOutput()
	}
}

func runIP(ctx context.Context, args ...string) (string, error) {
	out, err := exec.CommandContext(ctx, "ip", args...).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func enableIPForwarding(ctx context.Context) error {
	out, err := exec.CommandContext(ctx, "sysctl", "-w", "net.ipv4.ip_forward=1").CombinedOutput()
	if err != nil {
		return fmt.Errorf("enable IP forwarding: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

func removeMasquerade(ctx context.Context, guestCIDR string) {
	args := []string{"-t", "nat", "-D", "POSTROUTING", "-s", guestCIDR, "-j", "MASQUERADE"}
	_, _ = exec.CommandContext(ctx, "iptables", args...).CombinedOutput()
}

func addMasquerade(ctx context.Context, guestCIDR string) error {
	// Idempotent-ish: only add if not already present, to avoid piling up
	// duplicate MASQUERADE rules across VM restarts.
	check := exec.CommandContext(ctx, "iptables", "-t", "nat", "-C", "POSTROUTING", "-s", guestCIDR, "-j", "MASQUERADE")
	if err := check.Run(); err == nil {
		return nil
	}
	out, err := exec.CommandContext(ctx, "iptables", "-t", "nat", "-A", "POSTROUTING", "-s", guestCIDR, "-j", "MASQUERADE").CombinedOutput()
	if err != nil {
		return fmt.Errorf("add masquerade rule: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

func tapDeviceName(vmID string) string {
	id := strings.ReplaceAll(vmID, "-", "")
	if len(id) > 11 {
		id = id[:11]
	}
	return "fcs" + id // Linux interface names must stay under 16 chars.
}

// randomLocalMAC generates a locally-administered, unicast MAC address
// (the "02" prefix) so it never collides with vendor-assigned addresses.
func randomLocalMAC() (string, error) {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	buf[0] = (buf[0] & 0xfe) | 0x02
	return net.HardwareAddr(buf).String(), nil
}

// allocatePointToPoint deterministically derives a /30 host/guest IP pair
// from the VM ID within the given base network (default 172.16.0.0/16), so
// repeated calls for the same VM ID are stable and different VMs don't
// collide with reasonably high probability. This keeps allocation
// stateless: no lease table to persist or leak across restarts.
func allocatePointToPoint(vmID, base string) (hostIP, guestIP string, err error) {
	if base == "" {
		base = "172.16.0.0/16"
	}
	_, network, err := net.ParseCIDR(base)
	if err != nil {
		return "", "", fmt.Errorf("invalid base network %q: %w", base, err)
	}
	ones, bits := network.Mask.Size()
	if bits-ones < 4 {
		return "", "", fmt.Errorf("base network %q is too small for /30 allocation", base)
	}
	offset := hashToUint32(vmID) % (1 << uint(bits-ones-2))
	ip := network.IP.To4()
	if ip == nil {
		return "", "", fmt.Errorf("only IPv4 base networks are supported")
	}
	val := uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
	val += offset * 4
	host := val + 1
	guest := val + 2
	return uint32ToIP(host).String(), uint32ToIP(guest).String(), nil
}

func hashToUint32(s string) uint32 {
	var h uint32 = 2166136261
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return h
}

func networkBootArgs(bootArgs string, cfg NetworkConfig) string {
	if bootArgs == "" {
		bootArgs = "console=ttyS0 reboot=k panic=1 pci=off"
	}
	guestIP := strings.SplitN(cfg.GuestCIDR, "/", 2)[0]
	hostIP := strings.SplitN(cfg.HostCIDR, "/", 2)[0]
	return bootArgs + " ip=" + guestIP + "::" + hostIP + ":255.255.255.252::" + cfg.IfaceID + ":off"
}

func uint32ToIP(v uint32) net.IP {
	return net.IPv4(byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
}
