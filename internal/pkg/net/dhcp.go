/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package net

import (
	"log"
	"os"
	"time"

	"github.com/autonomy/dhcp/dhcpv4"
	"github.com/autonomy/dhcp/dhcpv4/client4"
	"github.com/autonomy/dhcp/netboot"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

func startDHCPClient() (err error) {
	//dhcp request
	modifiers := []dhcpv4.Modifier{
		dhcpv4.WithRequestedOptions(
			dhcpv4.OptionHostName,
			dhcpv4.OptionClasslessStaticRouteOption,
			dhcpv4.OptionDNSDomainSearchList,
			dhcpv4.OptionNTPServers,
		),
	}

	// Send hostname with DHCP request only if known and not an IP address
	if hostname, err := os.Hostname(); err == nil && !hostnameIsIP() {
		modifiers = append(modifiers, dhcpv4.WithOption(dhcpv4.OptHostName(hostname)))
	}

	if err = dhclient(ifname, modifiers); err != nil {
		return err
	}

	// Set up dhcp renewals every 5m
	go func() {
		for {
			// TODO pick this out of the dhclient/netconf response
			// so we can request less frequently
			time.Sleep(5 * time.Minute)
			log.Println("Renewing dhcp lease")
			if err = dhclient(ifname, modifiers); err != nil {
				// Probably need to do something better here but not sure there's much to do
				log.Println("Failed to renew dhcp lease, ", err)
			}
		}
	}()
	return nil
}

func dhclient(ifname string, modifiers []dhcpv4.Modifier) error {
	var err error
	var netconf *netboot.NetConf

	if netconf, err = dhclient4(ifname, modifiers...); err != nil {
		return err
	}
	if err = netboot.ConfigureInterface(ifname, netconf); err != nil {
		return err
	}

	return err
}

// nolint: gocyclo
func dhclient4(ifname string, modifiers ...dhcpv4.Modifier) (*netboot.NetConf, error) {
	attempts := 10
	client := client4.NewClient()
	var (
		conv []*dhcpv4.DHCPv4
		err  error
	)
	for attempt := 0; attempt < attempts; attempt++ {
		log.Printf("requesting DHCP lease: attempt %d of %d", attempt+1, attempts)
		conv, err = client.Exchange(ifname, modifiers...)
		if err != nil && attempt < attempts {
			log.Printf("failed to request DHCP lease: %v", err)
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}
		break
	}

	for _, m := range conv {
		if m.OpCode == dhcpv4.OpcodeBootReply && m.MessageType() == dhcpv4.MessageTypeOffer {
			if m.YourIPAddr != nil {
				log.Printf("using IP address %s", m.YourIPAddr.String())
			}

			if m.HostName() != "" {
				log.Printf("using hostname: %s", m.HostName())
				if err = unix.Sethostname([]byte(m.HostName())); err != nil {
					return nil, err
				}
			}

			break
		}
	}

	netconf, _, err := netboot.ConversationToNetconfv4(conv)
	if err != nil {
		return nil, err
	}

	return netconf, err
}
