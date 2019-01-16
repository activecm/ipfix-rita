package yaml

import "net"
import "github.com/pkg/errors"

//filtering implements config.Filtering
type filtering struct {
	AlwaysInclude   []string `yaml:"AlwaysInclude"`
	NeverInclude    []string `yaml:"NeverInclude"`
	InternalSubnets []string `yaml:"InternalSubnets"`
}

func (f *filtering) GetAlwaysIncludeSubnets() ([]net.IPNet, []error) {
	return f.parseSubnetList(f.AlwaysInclude)
}

func (f *filtering) GetNeverIncludeSubnets() ([]net.IPNet, []error) {
	return f.parseSubnetList(f.NeverInclude)
}

func (f *filtering) GetInternalSubnets() ([]net.IPNet, []error) {
	return f.parseSubnetList(f.InternalSubnets)
}

func (f *filtering) parseSubnetList(netList []string) ([]net.IPNet, []error) {
	var errorList []error
	var nets []net.IPNet
	for j := range netList {
		//parse as network
		_, network, err := net.ParseCIDR(netList[j])
		if err != nil {
			//parse as IP
			ipAddr := net.ParseIP(netList[j])

			if ipAddr == nil {
				errorList = append(errorList, errors.WithStack(err))
				continue
			}

			network = f.ipToIPNet(ipAddr)
		}

		nets = append(nets, *network)
	}
	return nets, errorList
}

func (f *filtering) ipToIPNet(ipAddr net.IP) *net.IPNet {
	var netmask net.IPMask
	if ipAddr.To4() == nil {
		netmask = net.CIDRMask(32, 32)
	} else {
		netmask = net.CIDRMask(128, 128)
	}
	return &net.IPNet{
		IP:   ipAddr,
		Mask: netmask,
	}
}
