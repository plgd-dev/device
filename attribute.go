package ocfsdk

type AttributeI interface {
	IdI
	GetValue(transaction TransactionI) (value interface{}, err error)
	SetValue(transaction TransactionI, value interface{}) error
}

type Attribute struct {
	Id
	Value ValueI
	Limit LimitI
}

func (a *Attribute) GetValue(transaction TransactionI) (interface{}, error) {
	if v, ok := a.Value.(ValueGetI); ok {
		return v.Get(transaction)
	}

	return nil, ErrAccessDenied
}

func (a *Attribute) SetValue(transaction TransactionI, value interface{}) error {
	if err := a.Limit.ValidateValue(value); err != nil {
		return err
	}
	if v, ok := a.Value.(ValueSetI); ok {
		return v.Set(transaction, value)
	}

	return ErrAccessDenied
}

func NewAttribute(id string, value ValueI, limit LimitI) (AttributeI, error) {

	if len(id) == 0 || value == nil || limit == nil {
		return nil, ErrInvalidParams
	}
	return &Attribute{Id: Id{id: id}, Value: value, Limit: limit}, nil
}
