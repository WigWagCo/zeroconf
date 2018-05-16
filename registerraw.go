package zeroconf

import (
	"fmt"
	"net"
	"os"
	"strings"
)

// RegisterServiceEntry let's you register a service by handing in a raw ServiceEntry
// struct for better flexibility, along with string array of ips to publish and intefaces
// to run the server on. notIfaces lets you black list specific interface to not publish to
func RegisterServiceEntry(entry *ServiceEntry, ips []string, ifaces []string, notIfaces []string) (*Server, error) {
	// entry := NewServiceEntry(instance, service, domain)
	// entry.Port = port
	// entry.Text = text
	// entry.HostName = host

	if entry.Instance == "" {
		return nil, fmt.Errorf("Missing service instance name")
	}
	if entry.Service == "" {
		return nil, fmt.Errorf("Missing service name")
	}
	// if entry.HostName == "" {
	// 	return nil, fmt.Errorf("Missing host name")
	// }
	if entry.Domain == "" {
		entry.Domain = "local"
	}
	if entry.Port == 0 {
		return nil, fmt.Errorf("Missing port")
	}

	var err error
	if entry.HostName == "" {
		entry.HostName, err = os.Hostname()
		if err != nil {
			return nil, fmt.Errorf("Could not determine host")
		}
	}

	if !strings.HasSuffix(trimDot(entry.HostName), entry.Domain) {
		entry.HostName = fmt.Sprintf("%s.%s.", trimDot(entry.HostName), trimDot(entry.Domain))
	}

	for _, ip := range ips {
		ipAddr := net.ParseIP(ip)
		if ipAddr == nil {
			return nil, fmt.Errorf("Failed to parse given IP: %v", ip)
		} else if ipv4 := ipAddr.To4(); ipv4 != nil {
			entry.AddrIPv4 = append(entry.AddrIPv4, ipAddr)
		} else if ipv6 := ipAddr.To16(); ipv6 != nil {
			entry.AddrIPv6 = append(entry.AddrIPv6, ipAddr)
		} else {
			return nil, fmt.Errorf("The IP is neither IPv4 nor IPv6: %#v", ipAddr)
		}
	}

	var _ifaces []net.Interface

	if len(ifaces) == 0 {
		_ifaces = listMulticastInterfaces()
	} else {
		for _, name := range ifaces {
			var iface net.Interface
			actualiface, err := net.InterfaceByName(name)
			if err == nil {
				iface = *actualiface
			} else {
				return nil, fmt.Errorf("interface not found %s", name)
			}
			_ifaces = append(_ifaces, iface)
		}
	}

	for _, name := range notIfaces {
	innerIfLoop:
		for i, ifs := range _ifaces {
			if ifs.Name == name {
				// this removes interface 'i' from the list
				// (bump the last interface off the list, and replace slot 'i'
				// then resize the slice, not have the - now duplicated - last slot)
				_ifaces[i] = _ifaces[len(_ifaces)-1]
				_ifaces[len(_ifaces)-1] = net.Interface{} // make sure GC cleanups all fields in Interface
				_ifaces = _ifaces[:len(_ifaces)-1]
				break innerIfLoop
			}
		}
	}

	s, err := newServer(_ifaces)
	if err != nil {
		return nil, err
	}

	s.service = entry
	go s.mainloop()
	go s.probe()

	return s, nil

}
