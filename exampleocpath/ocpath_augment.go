package exampleocpath

func (r *Root) WithName(name string) *Root {
	r.SetCustomDataKey("name", name)
	return r
}
