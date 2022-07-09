package corplink

import (
	"errors"
	"net"
	"os/exec"
	"reflect"
	"syscall"

	"golang.org/x/sys/unix"
)

var ioctlFD int
var interfaceIP net.IP

func loadFD() (err error) {
	if ioctlFD != 0 {
		return nil
	}
	ioctlFD, err = syscall.Socket(unix.AF_INET, unix.SOCK_DGRAM, 0)
	return err
}

func devName(name string) (devName [unix.IFNAMSIZ]byte) {
	copy(devName[:], name)
	return
}

type IfAliasReq struct {
	Name [unix.IFNAMSIZ]byte
	Addr unix.RawSockaddrInet4
}

type IfReq struct {
	Name [unix.IFNAMSIZ]byte
	Flag int
}

func SetInterfaceUp(name string, up bool) error {
	err := loadFD()
	if err != nil {
		return err
	}
	devName := devName(name)
	ifReq := IfReq{
		Name: devName,
	}
	// get flags
	err = unix.IoctlSetInt(ioctlFD, unix.SIOCGIFFLAGS, toInt(&ifReq))
	if err != nil {
		return err
	}
	if up {
		ifReq.Flag |= unix.IFF_UP
	} else {
		ifReq.Flag ^= unix.IFF_UP
	}
	// set up flag
	err = unix.IoctlSetInt(ioctlFD, unix.SIOCSIFFLAGS, toInt(&ifReq))
	return err
}

func SetInterfaceMTU(name string, mtu int) error {
	err := loadFD()
	if err != nil {
		return err
	}
	devName := devName(name)
	err = unix.IoctlSetIfreqMTU(ioctlFD, &unix.IfreqMTU{
		Name: devName,
		MTU:  int32(mtu),
	})
	return err
}

func SetInterfaceAddress(name, addr string) error {
	err := loadFD()
	if err != nil {
		return err
	}
	ip, _, err := net.ParseCIDR(addr)
	if err != nil {
		return err
	}
	interfaceIP = ip
	devName := devName(name)

	// set addr
	var req IfAliasReq
	if len(ip.To4()) == net.IPv4len {
		var realIP [4]byte
		copy(realIP[:], ip.To4())
		req = IfAliasReq{
			Name: devName,
			Addr: unix.RawSockaddrInet4{
				Len:    16,
				Family: unix.AF_INET,
				Addr:   realIP,
			},
		}
	} else {
		return errors.New("not support ipv6 for now")
	}
	err = unix.IoctlSetInt(ioctlFD, unix.SIOCSIFADDR, toInt(&req))
	if err != nil {
		println(err.Error())
		panic(err)
	}
	return nil
}

func toInt(data any) int {
	v := reflect.ValueOf(data)
	return int(v.Pointer())
}

func AddInterfaceRoute(name, network string) error {
	// TODO: replace with native implement like the others
	cmd := exec.Command("route", "add", "-net", network, interfaceIP.String())
	return cmd.Run()
}
