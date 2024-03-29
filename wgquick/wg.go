// nolint:errorlint,cyclop
package wgquick

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/getlantern/byteexec"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"go.xsfx.dev/wg-quicker/assets"
	"go.xsfx.dev/wg-quicker/tools/pidof"
	"golang.org/x/sys/unix"
	"golang.zx2c4.com/wireguard/wgctrl"
)

// userspace runs a embedded wireguard-go for interface creation.
func userspace(iface string) error {
	wgo, err := byteexec.New(assets.WGO, "wireguard-go")
	if err != nil {
		return fmt.Errorf("unable to create byteexec for wireguard-go: %w", err)
	}

	cmd := wgo.Command(iface)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("could not start wireguard-go: %w", err)
	}

	return nil
}

// rmWireguardGoSock removes sock file.
// This should remove a userspace wireguard-go implementation created interface.
func rmWireguardGoSock(iface string) error {
	sock := fmt.Sprintf("/var/run/wireguard/%s.sock", iface)

	if _, err := os.Stat(sock); os.IsNotExist(err) {
		return nil
	}

	if err := os.Remove(sock); err != nil {
		return fmt.Errorf("could not remove %s: %w", sock, err)
	}

	return nil
}

// Up sets and configures the wg interface. Mostly equivalent to `wg-quick up iface`.
func Up(cfg *Config, iface string, uspace bool, logger logrus.FieldLogger) error {
	log := logger.WithField("iface", iface)

	_, err := netlink.LinkByName(iface)
	if err == nil {
		return os.ErrExist
	}

	if _, ok := err.(netlink.LinkNotFoundError); !ok {
		return fmt.Errorf("%w", err)
	}

	for _, dns := range cfg.DNS {
		if err := execSh("resolvconf -a tun.%i -m 0 -x", iface, log, fmt.Sprintf("nameserver %s\n", dns)); err != nil {
			return err
		}
	}

	if cfg.PreUp != "" {
		if err := execSh(cfg.PreUp, iface, log); err != nil {
			return err
		}

		log.Infoln("applied pre-up command")
	}

	if err := Sync(cfg, iface, uspace, logger); err != nil {
		return err
	}

	if cfg.PostUp != "" {
		if err := execSh(cfg.PostUp, iface, log); err != nil {
			return err
		}

		log.Infoln("applied post-up command")
	}

	return nil
}

// Down destroys the wg interface. Mostly equivalent to `wg-quick down iface`.
func Down(cfg *Config, iface string, logger logrus.FieldLogger) error {
	log := logger.WithField("iface", iface)

	link, err := netlink.LinkByName(iface)
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	if len(cfg.DNS) > 1 {
		if err := execSh("resolvconf -d tun.%s", iface, log); err != nil {
			return err
		}
	}

	if cfg.PreDown != "" {
		if err := execSh(cfg.PreDown, iface, log); err != nil {
			return err
		}

		log.Infoln("applied pre-down command")
	}

	if err := netlink.LinkDel(link); err != nil {
		return fmt.Errorf("%w", err)
	}

	log.Infoln("link deleted")

	if cfg.PostDown != "" {
		if err := execSh(cfg.PostDown, iface, log); err != nil {
			return err
		}

		log.Infoln("applied post-down command")
	}

	// Some embedded userspace wireguard-go cleanup jobs.

	// Removes interface through sock file deletion.
	if err := rmWireguardGoSock(iface); err != nil {
		return fmt.Errorf("could not remove wireguard-go sock file: %w", err)
	}

	// Trying to kill embedded wireguard-go process.
	if err := pidof.Pkill(".byteexec/wireguard-go " + iface); err != nil {
		return fmt.Errorf("could not kill wireguard-go process: %w", err)
	}

	return nil
}

func execSh(command string, iface string, log logrus.FieldLogger, stdin ...string) error {
	cmd := exec.Command("sh", "-ce", strings.ReplaceAll(command, "%i", iface)) // nolint: gosec

	if len(stdin) > 0 {
		log = log.WithField("stdin", strings.Join(stdin, ""))
		b := &bytes.Buffer{}

		for _, ln := range stdin {
			if _, err := fmt.Fprint(b, ln); err != nil {
				return fmt.Errorf("%w", err)
			}
		}

		cmd.Stdin = b
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.WithError(err).Errorf("failed to execute %s:\n%s", cmd.Args, out)

		return fmt.Errorf("%w", err)
	}

	log.Infof("executed %s:\n%s", cmd.Args, out)

	return nil
}

// Sync the config to the current setup for given interface.
// It perform 4 operations:
// * SyncLink --> makes sure link is up and type wireguard.
// * SyncWireguardDevice --> configures allowedIP & other wireguard specific settings.
// * SyncAddress --> synces linux addresses bounded to this interface.
// * SyncRoutes --> synces all allowedIP routes to route to this interface.
func Sync(cfg *Config, iface string, uspace bool, logger logrus.FieldLogger) error {
	log := logger.WithField("iface", iface)

	link, err := SyncLink(cfg, iface, uspace, log)
	if err != nil {
		log.WithError(err).Errorln("cannot sync wireguard link")

		return fmt.Errorf("%w", err)
	}

	log.Info("synced link")

	if err := SyncWireguardDevice(cfg, link, log); err != nil {
		log.WithError(err).Errorln("cannot sync wireguard link")

		return err
	}

	log.Info("synced link")

	if err := SyncAddress(cfg, link, log); err != nil {
		log.WithError(err).Errorln("cannot sync addresses")

		return fmt.Errorf("%w", err)
	}

	log.Info("synced addresss")

	managedRoutes := make([]net.IPNet, 0)

	for _, peer := range cfg.Peers {
		managedRoutes = append(managedRoutes, peer.AllowedIPs...)
	}

	if err := SyncRoutes(cfg, link, managedRoutes, log); err != nil {
		log.WithError(err).Errorln("cannot sync routes")

		return fmt.Errorf("%w", err)
	}

	log.Info("synced routed")
	log.Info("successfully synced device")

	return nil
}

// SyncWireguardDevice synces wireguard vpn setting on the given link.
// It does not set routes/addresses beyond wg internal crypto-key routing, only handles wireguard specific settings.
func SyncWireguardDevice(cfg *Config, link netlink.Link, log logrus.FieldLogger) error {
	cl, err := wgctrl.New()
	if err != nil {
		log.WithError(err).Errorln("cannot setup wireguard device")

		return fmt.Errorf("%w", err)
	}

	if err := cl.ConfigureDevice(link.Attrs().Name, cfg.Config); err != nil {
		log.WithError(err).Error("cannot configure device")

		return fmt.Errorf("%w", err)
	}

	return nil
}

// SyncLink synces link state with the config.
// It does not sync Wireguard settings, just makes sure the device is up and type wireguard.
func SyncLink(cfg *Config, iface string, uspace bool, log logrus.FieldLogger) (netlink.Link, error) { //nolint:ireturn
	link, err := netlink.LinkByName(iface)
	// nolint: nestif
	if err != nil {
		if _, ok := err.(netlink.LinkNotFoundError); !ok {
			log.WithError(err).Error("cannot read link")

			return nil, fmt.Errorf("%w", err)
		}

		log.Info("link not found, creating")

		wgLink := &netlink.GenericLink{
			LinkAttrs: netlink.LinkAttrs{
				Name: iface,
				MTU:  cfg.MTU,
			},
			LinkType: "wireguard",
		}

		if uspace {
			log.Info("enforcing embedded wireguard-go")

			if err := userspace(iface); err != nil {
				log.WithError(err).Errorf("cannot create link through wireguard-go: %s", err.Error())

				return nil, fmt.Errorf("cannot create link: %w", err)
			}
		} else if !uspace {
			if err := netlink.LinkAdd(wgLink); err != nil {
				log.WithError(err).Errorf("cannot create link: %s", err.Error())
				log.Info("trying to use embedded wireguard-go")

				if err := userspace(iface); err != nil {
					log.WithError(err).Errorf("cannot create link through wireguard-go: %s", err.Error())

					return nil, fmt.Errorf("cannot create link: %w", err)
				}
			}
		}

		// Needs some sleeping to wait for interface creating.
		time.Sleep(1 * time.Second)

		link, err = netlink.LinkByName(iface)
		if err != nil {
			log.WithError(err).Error("cannot read link")

			return nil, fmt.Errorf("%w", err)
		}
	}

	if err := netlink.LinkSetUp(link); err != nil {
		log.WithError(err).Error("cannot set link up")

		return nil, fmt.Errorf("%w", err)
	}

	log.Info("set device up")

	return link, nil
}

// SyncAddress adds/deletes all lind assigned IPV4 addressed as specified in the config
// nolint: funlen, gosec, scopelint
func SyncAddress(cfg *Config, link netlink.Link, log logrus.FieldLogger) error {
	addrs, err := netlink.AddrList(link, syscall.AF_INET)
	if err != nil {
		log.Error(err, "cannot read link address")

		return fmt.Errorf("%w", err)
	}

	// nil addr means I've used it
	presentAddresses := make(map[string]netlink.Addr)

	for _, addr := range addrs {
		log.WithFields(map[string]interface{}{
			"addr":  fmt.Sprint(addr.IPNet),
			"label": addr.Label,
		}).Debugf("found existing address: %v", addr)

		presentAddresses[addr.IPNet.String()] = addr
	}

	for _, addr := range cfg.Address {
		log := log.WithField("addr", addr.String())
		_, present := presentAddresses[addr.String()]
		presentAddresses[addr.String()] = netlink.Addr{} // mark as present

		if present {
			log.Info("address present")

			continue
		}

		if err := netlink.AddrAdd(link, &netlink.Addr{
			IPNet: &addr,
			Label: cfg.AddressLabel,
		}); err != nil {
			if errors.Is(err, syscall.EEXIST) {
				log.WithError(err).Error("cannot add addr")

				return fmt.Errorf("%w", err)
			}
		}

		log.Info("address added")
	}

	for _, addr := range presentAddresses {
		if addr.IPNet == nil {
			continue
		}

		log := log.WithFields(map[string]interface{}{
			"addr":  addr.IPNet.String(),
			"label": addr.Label,
		})

		if err := netlink.AddrDel(link, &addr); err != nil {
			log.WithError(err).Error("cannot delete addr")

			return fmt.Errorf("%w", err)
		}

		log.Info("addr deleted")
	}

	return nil
}

func fillRouteDefaults(rt *netlink.Route) {
	// fill defaults
	if rt.Table == 0 {
		rt.Table = unix.RT_CLASS_MAIN
	}

	if rt.Protocol == 0 {
		rt.Protocol = unix.RTPROT_BOOT
	}

	if rt.Type == 0 {
		rt.Type = unix.RTN_UNICAST
	}
}

// SyncRoutes adds/deletes all route assigned IPV4 addressed as specified in the config
// nolint: funlen, gosec, scopelint
func SyncRoutes(cfg *Config, link netlink.Link, managedRoutes []net.IPNet, log logrus.FieldLogger) error {
	wantedRoutes := make(map[string][]netlink.Route, len(managedRoutes))

	presentRoutes, err := netlink.RouteList(link, syscall.AF_INET)
	if err != nil {
		log.Error(err, "cannot read existing routes")

		return fmt.Errorf("%w", err)
	}

	for _, rt := range managedRoutes {
		rt := rt // make copy
		log.WithField("dst", rt.String()).Debug("managing route")

		nrt := netlink.Route{
			LinkIndex: link.Attrs().Index,
			Dst:       &rt,
			Table:     cfg.Table,
			Protocol:  cfg.RouteProtocol,
			Priority:  cfg.RouteMetric,
		}
		fillRouteDefaults(&nrt)
		wantedRoutes[rt.String()] = append(wantedRoutes[rt.String()], nrt)
	}

	for _, rtLst := range wantedRoutes {
		for _, rt := range rtLst {
			rt := rt // make copy
			log := log.WithFields(map[string]interface{}{
				"route":    rt.Dst.String(),
				"protocol": rt.Protocol,
				"table":    rt.Table,
				"type":     rt.Type,
				"metric":   rt.Priority,
			})

			if err := netlink.RouteReplace(&rt); err != nil {
				log.WithError(err).Errorln("cannot add/replace route")

				return fmt.Errorf("%w", err)
			}

			log.Infoln("route added/replaced")
		}
	}

	checkWanted := func(rt netlink.Route) bool {
		for _, candidateRt := range wantedRoutes[rt.Dst.String()] {
			if rt.Equal(candidateRt) {
				return true
			}
		}

		return false
	}

	for _, rt := range presentRoutes {
		log := log.WithFields(map[string]interface{}{
			"route":    rt.Dst.String(),
			"protocol": rt.Protocol,
			"table":    rt.Table,
			"type":     rt.Type,
			"metric":   rt.Priority,
		})

		if !(rt.Table == cfg.Table || (cfg.Table == 0 && rt.Table == unix.RT_CLASS_MAIN)) {
			log.Debug("wrong table for route, skipping")

			continue
		}

		if !(rt.Protocol == cfg.RouteProtocol) {
			log.Infof("skipping route deletion, not owned by this daemon")

			continue
		}

		if checkWanted(rt) {
			log.Debug("route wanted, skipping deleting")

			continue
		}

		if err := netlink.RouteDel(&rt); err != nil {
			log.WithError(err).Error("cannot delete route")

			return fmt.Errorf("%w", err)
		}

		log.Info("route deleted")
	}

	return nil
}
