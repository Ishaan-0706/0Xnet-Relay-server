package main

import (
	"context"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/client"
	"github.com/multiformats/go-multiaddr"
)

func main() {
	ctx := context.Background()

	// 1. Create a client host
	h, err := libp2p.New()
	if err != nil {
		panic(err)
	}

	fmt.Println("Client Peer ID:", h.ID())

	// 2. Relay address (PASTE FROM SERVER)
	relayAddrStr := "/ip4/127.0.0.1/tcp/4001/p2p/12D3KooWP78VeM3RoYwnzJ2iM3itoNp6rLv8GTuUnpN853engtUF"

	relayMA, err := multiaddr.NewMultiaddr(relayAddrStr)
	if err != nil {
		panic(err)
	}

	relayInfo, err := peer.AddrInfoFromP2pAddr(relayMA)
	if err != nil {
		panic(err)
	}

	// 3. Connect to relay
	if err := h.Connect(ctx, *relayInfo); err != nil {
		panic(err)
	}
	fmt.Println("✅ Connected to relay")

	// 4. Reserve a relay slot
	_, err = client.Reserve(ctx, h, *relayInfo)
	if err != nil {
		panic(err)
	}
	fmt.Println("📡 Relay reservation successful")

	const KeepAliveProtocol = "/keepalive/1.0.0"

	h.SetStreamHandler(KeepAliveProtocol, func(s network.Stream) {
		fmt.Println("🔗 Keepalive stream opened from", s.Conn().RemotePeer())
		defer s.Close()

		// Block forever (or until context cancel)
		<-server.ctx.Done()
	})

	// Keep running
	select {
	case <-time.After(time.Hour):
	}
}
