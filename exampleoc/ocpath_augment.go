package exampleoc

func (r *DevicePath) WithName(name string) *DevicePath {
	r.PutCustomData("name", name)
	return r
}
