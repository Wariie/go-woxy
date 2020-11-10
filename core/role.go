package core

type Role struct {
	module string
	name   string
	key    string
}

func (r *Role) setModule(moduleName string) {
	r.module = moduleName
}

func (r *Role) setKey(key string) {
	r.key = key
}

func (r *Role) setName(name string) {
	r.name = name
}
