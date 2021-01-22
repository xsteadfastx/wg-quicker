package main

import "go.xsfx.dev/wg-quicker/cmd"

//go:generate go-bindata -pkg assets -o assets/bindata.go -nomemcopy third_party/wireguard-go/wireguard-go

func main() {
	cmd.Execute()
}
