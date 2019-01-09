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
		_, net, err := net.ParseCIDR(netList[j])
		if err != nil {
			errorList = append(errorList, errors.WithStack(err))
		} else {
			nets = append(nets, *net)
		}
	}
	return nets, errorList
}
