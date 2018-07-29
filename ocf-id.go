package ocfsdk

type OCFId struct {
	id string
}

func (i *OCFId) GetId() string {
	return i.id
}
