package client

import (
	"fmt"
	"github.com/sailpoint/atlas-go/atlas"
	"github.com/sailpoint/atlas-go/atlas/beacon"
)

type beaconServiceLocator struct {
	beaconRegistrar beacon.Registrar
	delegate ServiceLocator
}

func NewBeaconServiceLocator(delegate ServiceLocator, beaconRegistrar beacon.Registrar) *beaconServiceLocator {
	l := &beaconServiceLocator{}
	l.delegate = delegate
	l.beaconRegistrar = beaconRegistrar

	return l
}

func (l *beaconServiceLocator) GetURL(org atlas.Org, service string) string {
	registration, err := l.beaconRegistrar.FindByTenantAndService(beacon.TenantID(org), beacon.ServiceID(service))
	if err != nil || registration == nil {
		return l.delegate.GetURL(org, service)
	}
	return fmt.Sprintf("http://%s",registration.Connection)
}
