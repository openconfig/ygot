package exampleocpath

func (r *Root) WithName(name string) *Root {
	r.CustomData["name"] = name
	return r
}
