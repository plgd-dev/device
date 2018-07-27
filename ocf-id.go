package ocfsdk

type OCFId struct {
	Id string
}

func (i *OCFId) GetId() string {
	return i.Id
}
