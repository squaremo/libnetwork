package remote

import (
	"errors"
	"net"

	"github.com/docker/libnetwork/driverapi"
	"github.com/docker/libnetwork/sandbox"
	"github.com/docker/libnetwork/types"

	"github.com/docker/docker/plugins"
)

var errNoCallback = errors.New("No Callback handler registered with Driver")

type driver struct {
	networkType string
	client      *plugins.Client
}

// New constructs a fresh remote driver
func New(networkType string, client *plugins.Client) driverapi.Driver {
	return &driver{
		networkType,
		client,
	}
}

func Init(dc driverapi.DriverCallback) error {
	return nil
}

func (d *driver) Config(option map[string]interface{}) error {
	return driverapi.ErrNotImplemented
}

type createNetwork struct {
	Id      string
	Options map[string]interface{}
}

func (d *driver) CreateNetwork(id types.UUID, option map[string]interface{}) error {
	return d.client.Call("NetworkDriver.CreateNetwork", &createNetwork{string(id), option}, nil)
}

func (d *driver) DeleteNetwork(nid types.UUID) error {
	return driverapi.ErrNotImplemented
}

type createEndpoint struct {
	NetworkId string
	Id        string
	Options   map[string]interface{}
}

type endpointInterface struct {
	SrcName     string
	DstName     string
	Address     string
	AddressIPv6 string
}

type createEndpointResponse struct {
	Interfaces  []*endpointInterface
	Gateway     string
	GatewayIPv6 string
}

// TODO: IPv6, Gateway etc.
func (r *createEndpointResponse) toSandboxInfo() (*sandbox.Info, error) {
	var (
		ifaces []*sandbox.Interface = make([]*sandbox.Interface, len(r.Interfaces))
	)
	for i, inIf := range r.Interfaces {
		outIf := &sandbox.Interface{
			SrcName: inIf.SrcName,
			DstName: inIf.DstName,
		}
		ip, ipnet, err := net.ParseCIDR(inIf.Address)
		if err != nil {
			return nil, err
		}
		ipnet.IP = ip
		outIf.Address = ipnet
		ifaces[i] = outIf
	}
	return &sandbox.Info{
		Interfaces:  ifaces,
		Gateway:     nil,
		GatewayIPv6: nil,
	}, nil
}

func (d *driver) CreateEndpoint(nid, eid types.UUID, epOptions map[string]interface{}) (*sandbox.Info, error) {
	var res createEndpointResponse

	create := &createEndpoint{
		string(nid),
		string(eid),
		epOptions,
	}

	if err := d.client.Call("NetworkDriver.CreateEndpoint", create, &res); err != nil {
		return nil, err
	}
	if info, err := res.toSandboxInfo(); err != nil {
		return nil, err
	} else {
		return info, nil
	}
}

type deleteEndpoint struct {
	NetworkId  string
	EndpointId string
}

func (d *driver) DeleteEndpoint(nid, eid types.UUID) error {
	return d.client.Call("NetworkDriver.DeleteEndpoint", &deleteEndpoint{string(nid), string(eid)}, nil)
}

func (d *driver) EndpointInfo(nid, eid types.UUID) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

type join struct {
	NetworkId  string
	EndpointId string
	SandboxKey string
	Options    map[string]interface{}
}

// Join method is invoked when a Sandbox is attached to an endpoint.
func (d *driver) Join(nid, eid types.UUID, sboxKey string, options map[string]interface{}) (*driverapi.JoinInfo, error) {
	var info driverapi.JoinInfo
	if err := d.client.Call("NetworkDriver.Join", &join{string(nid), string(eid), sboxKey, options}, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

type leave struct {
	NetworkId  string
	EndpointId string
	Options    map[string]interface{}
}

// Leave method is invoked when a Sandbox detaches from an endpoint.
func (d *driver) Leave(nid, eid types.UUID, options map[string]interface{}) error {
	return d.client.Call("NetworkDriver.Leave", &leave{string(nid), string(eid), options}, nil)
}

func (d *driver) Type() string {
	return d.networkType
}
