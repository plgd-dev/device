package ocfsdk

// Error errors type of main
type Error string

func (e Error) Error() string { return string(e) }

// ErrInvalidType value has invalid type
const ErrInvalidType = Error("Invalid type")

// ErrInvalidEnumValue value doesn't belongs to enum
const ErrInvalidEnumValue = Error("Invalid enum value")

// ErrUnprovidedKeyOfMap value contains map with unknown key
const ErrUnprovidedKeyOfMap = Error("Invalid key of map")

// ErrAccessDenied cannot access to object
const ErrAccessDenied = Error("Access denied")

// ErrInvalidInterface interface is not valid
const ErrInvalidInterface = Error("Invalid interface")

// ErrOperationNotSupported operation is not supported
const ErrOperationNotSupported = Error("Operation is not supported")

// ErrInvalidParams invalid parameters
const ErrInvalidParams = Error("Invalid params")

// ErrInvalidIterator invalid iterator
const ErrInvalidIterator = Error("Invalid iterator")

// ErrExist object already exist
const ErrExist = Error("object already exist")

// ErrNotExist objest doesn't exist
const ErrNotExist = Error("object doesn't exist")
