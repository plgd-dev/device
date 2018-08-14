package ocfsdk

import (
	"fmt"
)

const (
	discoveryURI          = "/oic/res"
	discoveryResourceType = "oic.wk.res"
)

type resourceDiscoveryInterface struct {
	resourceInterface
}

func (ri *resourceDiscoveryInterface) Retrieve(req RequestI, transaction TransactionI) (PayloadI, error) {
	discovery := make([]interface{}, 0)
	di, err := req.GetDevice().GetDeviceID()
	if err != nil {
		return nil, err
	}
	for resIt := req.GetDevice().NewResourceIterator(); resIt.Value() != nil; resIt.Next() {
		res := make(map[string]interface{})

		res["href"] = resIt.Value().GetID()
		if resIt.Value().GetID() == discoveryURI {
			res["rel"] = "self"
			res["anchor"] = fmt.Sprintf("ocf://%s%s", di, discoveryURI)
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
			if it.Value().GetID() != "" {
				iface = append(iface, it.Value().GetID())
			}
		}
		res["if"] = iface

		rt := make([]string, 0)
		for it := resIt.Value().NewResourceTypeIterator(); it.Value() != nil; it.Next() {
			rt = append(rt, it.Value().GetID())
		}
		res["rt"] = rt
		errors := make(map[string]error)
		for it := req.GetResource().NewResourceTypeIterator(); it.Value() != nil; it.Next() {
			for itA := it.Value().NewAttributeIterator(); itA.Value() != nil; itA.Next() {
				if value, err := itA.Value().GetValue(transaction); err == nil {
					res[itA.Value().GetID()] = value
				} else {
					errors[itA.Value().GetID()] = err
				}
			}
		}
		discovery = append(discovery, res)
	}
	return discovery, nil
}

func newResourceDiscoveryInterface(name string) ResourceInterfaceI {
	return &resourceDiscoveryInterface{resourceInterface: resourceInterface{id: id{id: name}}}
}

type resourceDiscovery struct {
	ResourceMiddleware
}

//NewResourceDiscovery creates a resource discovery
func NewResourceDiscovery() (ResourceDiscoveryI, error) {

	rt, err := NewResourceType(discoveryResourceType, []AttributeI{})
	if err != nil {
		return nil, err
	}

	resourceParams := &ResourceParams{
		id:                 discoveryURI,
		Discoverable:       true,
		Observeable:        true,
		ResourceTypes:      []ResourceTypeI{rt},
		ResourceOperations: NewResourceOperationRetrieve(func() (TransactionI, error) { return &transactionDummy{}, nil }),
		ResourceInterfaces: []ResourceInterfaceI{newResourceDiscoveryInterface(""), newResourceDiscoveryInterface("oic.if.ll")},
	}

	resMid, err := NewResource(resourceParams)
	if err != nil {
		return nil, err
	}

	return &resourceDiscovery{ResourceMiddleware: ResourceMiddleware{resource: resMid}}, nil
}
