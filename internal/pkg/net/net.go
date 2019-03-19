/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package net

import (
	"net"
)

// IPAddrs finds and returns a list of non-loopback IPv4 addresses of the
// current machine.
func IPAddrs() (ips []net.IP, err error) {
	ips = []net.IP{}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return
	}

	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ips = append(ips, ipnet.IP)

			}
		}
	}

	return ips, nil
}

// Setup creates the network.
func Setup() (err error) {
	//ifup lo
	ifname := "lo"
	link, err := netlink.LinkByName(ifname)
	if err != nil {
		return err
	}
	if err = netlink.LinkSetUp(link); err != nil {
		return err
	}

	//ifup eth0
	ifname = "eth0"
	link, err = netlink.LinkByName(ifname)
	if err != nil {
		return err
	}
	if err = netlink.LinkSetUp(link); err != nil {
		return err
	}

	return startDHCPClient()
}

// Checks if the hostname matches the IP address of any interface
func hostnameIsIP() (bool, err error) {
	var hostname string
	if hostname, err = os.Hostname(); err != nil {
		return false, err
	}
	var ips []net.IP
	if ips, err = IPAdders(); err != nil {
		return false, err
	}
	for _, ip := range ips {
		if hostname == ip {
			return true, nil
		}
	}
	return false, nil
}
