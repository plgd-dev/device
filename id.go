package ocfsdk

type Id struct {
	id string
}

func (i *Id) GetId() string {
	return i.id
}
