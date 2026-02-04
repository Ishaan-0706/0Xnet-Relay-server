package main

import (
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/client"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	libp2ptls "github.com/libp2p/go-libp2p/p2p/security/tls"
	"github.com/multiformats/go-multiaddr"
)

func main() {
	relayAddr := flag.String("relay", "", "Relay server multiaddr")
	name := flag.String("name", "test-peer", "Peer name")
	flag.Parse()

	if *relayAddr == "" {
		fmt.Println("Usage: go run peer_test.go -relay <relay-multiaddr>")
		fmt.Println("Example: go run peer_test.go -relay /ip4/x.x.x.x/tcp/4001/p2p/QmXXX")
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fmt.Println("🚀 Starting test peer...")

	// Generate key
	privKey, _, _ := crypto.GenerateEd25519Key(rand.Reader)

	// Parse relay address
	relayMA, err := multiaddr.NewMultiaddr(*relayAddr)
	if err != nil {
		fmt.Printf("❌ Invalid relay address: %v\n", err)
		os.Exit(1)
	}

	relayInfo, err := peer.AddrInfoFromP2pAddr(relayMA)
	if err != nil {
		fmt.Printf("❌ Could not parse relay peer info: %v\n", err)
		os.Exit(1)
	}

	// Create host
	h, err := libp2p.New(
		libp2p.ListenAddrs(multiaddr.StringCast("/ip4/0.0.0.0/tcp/0")),
		libp2p.Identity(privKey),
		libp2p.Security(libp2ptls.ID, libp2ptls.New),
		libp2p.Security(noise.ID, noise.New),
		libp2p.EnableRelay(),
		libp2p.EnableHolePunching(),
	)
	if err != nil {
		fmt.Printf("❌ Failed to create host: %v\n", err)
		os.Exit(1)
	}
	defer h.Close()

	fmt.Printf("✅ Created peer: %s\n", h.ID().String()[:16])
	fmt.Printf("📍 Local addresses:\n")
	for _, addr := range h.Addrs() {
		fmt.Printf("   %s\n", addr)
	}

	// Connection notifications
	h.Network().Notify(&network.NotifyBundle{
		ConnectedF: func(n network.Network, conn network.Conn) {
			fmt.Printf("🔗 Connected to: %s\n", conn.RemotePeer().String()[:16])
		},
		DisconnectedF: func(n network.Network, conn network.Conn) {
			fmt.Printf("🔌 Disconnected from: %s\n", conn.RemotePeer().String()[:16])
		},
	})

	// Connect to relay
	fmt.Printf("\n📡 Connecting to relay: %s...\n", relayInfo.ID.String()[:16])

	connectCtx, connectCancel := context.WithTimeout(ctx, 30*time.Second)
	defer connectCancel()

	if err := h.Connect(connectCtx, *relayInfo); err != nil {
		fmt.Printf("❌ Failed to connect to relay: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ Connected to relay!")

	// Reserve slot on relay
	fmt.Println("📝 Reserving slot on relay...")

	reservation, err := client.Reserve(ctx, h, *relayInfo)
	if err != nil {
		fmt.Printf("❌ Failed to reserve relay slot: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Successfully reserved relay slot!")
	fmt.Printf("⏰ Reservation expires: %s\n", reservation.Expiration.Format(time.RFC3339))

	// Print relay address that other peers can use
	relayedAddr := fmt.Sprintf("%s/p2p-circuit/p2p/%s", *relayAddr, h.ID().String())
	fmt.Printf("\n🔗 Your relayed address (share this with other peers):\n")
	fmt.Printf("   %s\n", relayedAddr)

	// Keep alive and show status
	fmt.Println("\n✅ Peer is running! Press Ctrl+C to exit.")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Status ticker
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				peers := h.Network().Peers()
				fmt.Printf("📊 Status: %d connected peers\n", len(peers))
			}
		}
	}()

	// Wait for signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\n👋 Shutting down...")
}
