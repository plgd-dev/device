package ocfsdk

import (
	"fmt"
)

const (
	DISCOVERY_URI           = "/oic/res"
	DISCOVERY_RESOURCE_TYPE = "oic.wk.res"
)

type ResourceDiscoveryInterface struct {
	ResourceInterface
}

func (ri *ResourceDiscoveryInterface) Retrieve(req RequestI, transaction TransactionI) (PayloadI, error) {
	discovery := make([]interface{}, 0)
	di, err := req.GetDevice().GetDeviceId()
	if err != nil {
		return nil, err
	}
	for resIt := req.GetDevice().NewResourceIterator(); resIt.Value() != nil; resIt.Next() {
		res := make(map[string]interface{})

		res["href"] = resIt.Value().GetId()
		if resIt.Value().GetId() == DISCOVERY_URI {
			res["rel"] = "self"
			res["anchor"] = fmt.Sprintf("ocf://%s%s", di, DISCOVERY_URI)
		} else {
			res["anchor"] = fmt.Sprintf("ocf://%s", di)
		}
		bm := 0
		if resIt.Value().IsDiscoverable() {
			bm += 0x1
		}
		if resIt.Value().IsObserveable() {
			bm += 0x2
		}

		res["p"] = map[string]interface{}{
			"bm": bm,
			//TODO: "sec":
		}

		//TODO: res["eps"]

		iface := make([]string, 0)
		for it := resIt.Value().NewResourceInterfaceIterator(); it.Value() != nil; it.Next() {
			if it.Value().GetId() != "" {
				iface = append(iface, it.Value().GetId())
			}
		}
		res["if"] = iface

		rt := make([]string, 0)
		for it := resIt.Value().NewResourceTypeIterator(); it.Value() != nil; it.Next() {
			rt = append(rt, it.Value().GetId())
		}
		res["rt"] = rt
		errors := make(map[string]error)
		for it := req.GetResource().NewResourceTypeIterator(); it.Value() != nil; it.Next() {
			for itA := it.Value().NewAttributeIterator(); itA.Value() != nil; itA.Next() {
				if value, err := itA.Value().GetValue(transaction); err == nil {
					res[itA.Value().GetId()] = value
				} else {
					errors[itA.Value().GetId()] = err
				}
			}
		}
		discovery = append(discovery, res)
	}
	return discovery, nil
}

func newResourceDiscoveryInterface(name string) ResourceInterfaceI {
	return &ResourceDiscoveryInterface{ResourceInterface: ResourceInterface{Id: Id{id: name}}}
}

type ResourceDiscovery struct {
	ResourceMiddleware
}

func NewResourceDiscovery() (ResourceI, error) {

	rt, err := NewResourceType(DISCOVERY_RESOURCE_TYPE, []AttributeI{})
	if err != nil {
		return nil, err
	}

	resourceParams := &ResourceParams{
		Id:                 DISCOVERY_URI,
		Discoverable:       true,
		Observeable:        true,
		ResourceTypes:      []ResourceTypeI{rt},
		ResourceOperations: NewResourceOperationRetrieve(func() (TransactionI, error) { return &DummyTransaction{}, nil }),
		ResourceInterfaces: []ResourceInterfaceI{newResourceDiscoveryInterface(""), newResourceDiscoveryInterface("oic.if.ll")},
	}

	resMid, err := NewResource(resourceParams)
	if err != nil {
		return nil, err
	}

	return &ResourceDevice{ResourceMiddleware: ResourceMiddleware{resource: resMid}}, nil
}
