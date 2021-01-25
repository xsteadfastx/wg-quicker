<p align="center">
        <img src="./logo.png" width="587" alt="logo">
        <p align="center"><b>wg-quick like library with batteries included</b></p>
        <p align="center">
                <a href="https://ci.xsfx.dev/xsteadfastx/wg-quicker"><img src="https://ci.xsfx.dev/api/badges/xsteadfastx/wg-quicker/status.svg" /></a>
                <a href="https://pkg.go.dev/go.xsfx.dev/wg-quicker"><img src="https://pkg.go.dev/badge/go.xsfx.dev/wg-quicker.svg" alt="Go Reference"></a>
                <a href="https://goreportcard.com/report/go.xsfx.dev/wg-quicker"><img src="https://goreportcard.com/badge/go.xsfx.dev/wg-quicker" alt="Go Report Card"></a>
        </p>
</p>

---

This is a friendly fork of [wg-quick-go](https://github.com/nmiculinic/wg-quick-go). It contains everything needed to get a system into a wireguard vpn network. If there is no wireguard kernel modul available, it will spin up the embedded wireguard-go to create a wireguard interface.

# Installation

## Prebuild packages

Get these on release [page](https://git.xsfx.dev/xsteadfastx/wg-quicker/releases).

## Compile it for yourself

- `git clone https://git.xsfx.dev/xsteadfastx/wg-quicker.git`
- `cd wg-quicker`
- `make build`

# Roadmap

- [x] full wg-quick feature parity
  - [x] PreUp
  - [x] PostUp
  - [x] PreDown
  - [x] PostDown
  - [x] DNS
  - [x] MTU
  - [x] Save --> Use MarshallText interface to save config
- [x] Sync
- [x] Up
- [x] Down
- [x] MarshallText
- [x] UnmarshallText
- [x] Minimal test
- [x] Embedded [wireguard-go](https://git.zx2c4.com/wireguard-go/about/)
- [ ] Integration tests ((TODO; have some virtual machines/kvm and wreck havoc :) ))

# Caveats

- Endpoints DNS MarshallText is unsupported
- Pre/Post Up/Down doesn't support escaped `%i`, that is all `%i` are expanded to interface name.
- SaveConfig in config is only a placeholder (( since there's no reading/writing from files )). Use Unmarshall/Marshall Text to save/load config (( you're responsible for IO)).
