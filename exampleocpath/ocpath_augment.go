package exampleocpath

func (r *Root) WithName(name string) *Root {
	r.PutCustomData("name", name)
	return r
}
