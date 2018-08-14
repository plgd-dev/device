package ocfsdk

type id struct {
	id string
}

func (i *id) GetID() string {
	return i.id
}
