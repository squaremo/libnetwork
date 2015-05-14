package remote

import (
	"errors"

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

func (d *driver) CreateNetwork(id types.UUID, option map[string]interface{}) error {
	return driverapi.ErrNotImplemented
}

func (d *driver) DeleteNetwork(nid types.UUID) error {
	return driverapi.ErrNotImplemented
}

func (d *driver) CreateEndpoint(nid, eid types.UUID, epOptions map[string]interface{}) (*sandbox.Info, error) {
	return nil, driverapi.ErrNotImplemented
}

func (d *driver) DeleteEndpoint(nid, eid types.UUID) error {
	return driverapi.ErrNotImplemented
}

func (d *driver) EndpointInfo(nid, eid types.UUID) (map[string]interface{}, error) {
	return nil, driverapi.ErrNotImplemented
}

// Join method is invoked when a Sandbox is attached to an endpoint.
func (d *driver) Join(nid, eid types.UUID, sboxKey string, options map[string]interface{}) (*driverapi.JoinInfo, error) {
	return nil, driverapi.ErrNotImplemented
}

// Leave method is invoked when a Sandbox detaches from an endpoint.
func (d *driver) Leave(nid, eid types.UUID, options map[string]interface{}) error {
	return driverapi.ErrNotImplemented
}

func (d *driver) Type() string {
	return d.networkType
}
