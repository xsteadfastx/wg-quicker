// Mostly taken from: https://github.com/WireGuard/wgctrl-go/blob/master/cmd/wgctrl/main.go
//
// nolint: forbidigo, godox
package wgquick

import (
	"fmt"
	"net"
	"strings"

	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func Show(iface string) error {
	c, err := wgctrl.New()
	if err != nil {
		return fmt.Errorf("failed to open wgctrl: %w", err)
	}
	defer c.Close()

	var devices []*wgtypes.Device

	if iface != "" {
		d, err := c.Device(iface)
		if err != nil {
			return fmt.Errorf("failed to get device %s: %w", iface, err)
		}

		devices = append(devices, d)
	} else {
		devices, err = c.Devices()
		if err != nil {
			return fmt.Errorf("failed to get devices: %w", err)
		}
	}

	for _, d := range devices {
		printDevice(d)

		for _, p := range d.Peers {
			printPeer(p)
		}
	}

	return nil
}

func printDevice(d *wgtypes.Device) {
	const f = `interface: %s (%s)
  public key: %s
  private key: (hidden)
  listening port: %d

`

	fmt.Printf(
		f,
		d.Name,
		d.Type.String(),
		d.PublicKey.String(),
		d.ListenPort)
}

func printPeer(p wgtypes.Peer) {
	const f = `peer: %s
  endpoint: %s
  allowed ips: %s
  latest handshake: %s
  transfer: %d B received, %d B sent

`

	fmt.Printf(
		f,
		p.PublicKey.String(),
		// TODO(mdlayher): get right endpoint with getnameinfo.
		p.Endpoint.String(),
		ipsString(p.AllowedIPs),
		p.LastHandshakeTime.String(),
		p.ReceiveBytes,
		p.TransmitBytes,
	)
}

func ipsString(ipns []net.IPNet) string {
	ss := make([]string, 0, len(ipns))
	for _, ipn := range ipns {
		ss = append(ss, ipn.String())
	}

	return strings.Join(ss, ", ")
}
