package types

// Base device resource types.
const (
	Device = "oic.wk.d"
)

// Device resource types with special handling.
var (
	BaseTypes = []string{
		Device,
	}
	SpecialTypes = []string{
		"oic.r.doxm",
		"oic.r.pstat",
		"oic.r.acl2",
		"oic.r.cred",
		"oic.wk.introspection",
		"oic.wk.con",
		"oic.wk.p",
	}
)
