package exampleocpath

func (r *Root) WithName(name string) *Root {
	r.customData["name"] = name
	return r
}
