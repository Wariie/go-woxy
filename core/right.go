package core

//Entity -
type Entity struct {
	id    int
	name  string
	roles []Role
	key   string
}

func (e *Entity) setKey(key string) {
	e.key = key
}

//Role - Module Role
type Role struct {
	moduleName string
	name       string
}

func (r *Role) setModuleName(moduleName string) {
	r.moduleName = moduleName
}

func (r *Role) setName(name string) {
	r.name = name
}
