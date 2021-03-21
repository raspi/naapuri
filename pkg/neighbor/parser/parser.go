package parser

import (
	"encoding/binary"
	"fmt"
	"github.com/raspi/naapuri/pkg/neighbor/families"
	"github.com/raspi/naapuri/pkg/neighbor/state"
	"golang.org/x/sys/unix"
	"io"
	"net"
	"unsafe"
)

var nativeEndian binary.ByteOrder

func init() {
	buf := [2]byte{}
	*(*uint16)(unsafe.Pointer(&buf[0])) = uint16(0xABCD)

	switch buf {
	case [2]byte{0xCD, 0xAB}:
		nativeEndian = binary.LittleEndian
	case [2]byte{0xAB, 0xCD}:
		nativeEndian = binary.BigEndian
	default:
		panic("Could not determine native endianness.")
	}
}

type commonRawHeader struct {
	AddressFamily  uint32 // unix.AF_INET (IPv4), unix.AF_INET6 (IPv6), ...
	InterfaceIndex uint32 // Network Interface Card's index number (for example: eth0 = 5)
	State          uint16 // State
	Unknown        uint16 // flags??
	Unknown2       uint16 // ??
	Unknown3       uint16 // ??
}

type rawHeaderIPv4 struct {
	IPAddress  [4]uint8 // 4 bytes (A.B.C.D)
	Unknown1   uint32   // ??
	MACAddress [6]uint8 // 6 bytes (AA:BB:CC:DD:EE:FF)
	//Unknown2   [30]uint8 // ??
}

type ChangeEvent struct {
	Family          families.AddressFamily `json:"af"`      // IPv4 / IPv6
	InterfaceNumber int                    `json:"if_no"`   // for example 5
	InterfaceName   string                 `json:"if_name"` // for example eth0
	IPAddress       net.IP                 `json:"ip"`      // A.B.C.D / A:B:C:D:E:F:G:H
	MACAddress      net.HardwareAddr       `json:"mac"`     // Ethernet hardware address AA:BB:CC:DD:EE:FF
	State           state.State            `json:"state"`   // MAC address can be reached, etc
}

func (n ChangeEvent) String() string {
	return fmt.Sprintf(`%s #%d %s %s:%s %s`,
		n.InterfaceName, n.InterfaceNumber, n.MACAddress, n.Family, n.IPAddress, n.State)
}

// Parse parses raw NetLink neighbor binary change messages
func Parse(rdr io.Reader) (che ChangeEvent, err error) {
	var rawheader commonRawHeader
	err = binary.Read(rdr, binary.LittleEndian, &rawheader)
	if err != nil {
		return che, fmt.Errorf(`error while marshalling header: %w`, err)
	}

	// Init event
	che = ChangeEvent{
		Family:          families.UnknownAddressFamily,
		State:           state.UnknownState,
		InterfaceNumber: -1,
		InterfaceName:   ``,
	}

	switch rawheader.AddressFamily {
	case unix.AF_INET: // IPv4
		che.Family = families.IPv4
	case unix.AF_INET6: // IPv4
		che.Family = families.IPv6
	default:
		return che, fmt.Errorf(`unknown address family 0x%[1]x %[1]d`, rawheader.AddressFamily)
	}

	che.InterfaceNumber = int(rawheader.InterfaceIndex)
	iface, err := net.InterfaceByIndex(che.InterfaceNumber)
	if err != nil {
		return che, fmt.Errorf(`could not get interface name: %w`, err)
	}

	if iface == nil {
		return che, fmt.Errorf(`nil interface in index %d`, che.InterfaceNumber)
	}

	che.InterfaceName = iface.Name

	switch rawheader.State {
	case 0x01:
		che.State = state.Incomplete
	case 0x02:
		che.State = state.Reachable
	case 0x04:
		che.State = state.Stale
	case 0x08:
		che.State = state.Delay
	case 0x10:
		che.State = state.Probe
	case 0x20:
		che.State = state.Failed
	case 0x40:
		che.State = state.NoARP
	case 0x80:
		che.State = state.Permanent
	case 0x00:
		che.State = state.None
	default:
		return che, fmt.Errorf(`unknown change state %v`, rawheader.State)
	}

	switch che.Family {
	case families.IPv4: // IPv4
		var raw rawHeaderIPv4

		err = binary.Read(rdr, nativeEndian, &raw)
		if err != nil {
			return che, fmt.Errorf(`could not marshal ipv4 message: %w`, err)
		}

		che.IPAddress = raw.IPAddress[:]
		che.MACAddress = raw.MACAddress[:]

		return che, nil
	case families.IPv6: // IPv6
		// TODO
		return che, fmt.Errorf(`ipv6 not yet supported`)
	} // /switch

	return che, nil
}
