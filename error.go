package ocfsdk

// Error errors type of main
type Error string

func (e Error) Error() string { return string(e) }

// ErrInvalidType value has invalid type
const ErrInvalidType = Error("Invalid type")

// ErrInvalidEnumValue value doesn't belongs to enum
const ErrInvalidEnumValue = Error("Invalid enum value")

// ErrInvalidKeyOfMap value contains map with unknown key
const ErrInvalidKeyOfMap = Error("Invalid key of map")

// ErrAccessDenied cannot access to object
const ErrAccessDenied = Error("Access denied")

// ErrInvalidInterface interfaec is not valid
const ErrInvalidInterface = Error("Invalid interface")

// ErrOperationNotSupported operation is not supported
const ErrOperationNotSupported = Error("Operation is not supported")

// ErrInvalidParams invalid parameters
const ErrInvalidParams = Error("Invalid params")

// ErrInvalidIterator invalid iterator
const ErrInvalidIterator = Error("Invalid iterator")
