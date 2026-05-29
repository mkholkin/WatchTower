package mapper

type Mapper[From any, To any, Key comparable] interface {
	Register(Key, func(f From) (To, error))
	Convert(Key, From) (To, error)
}

type mapRegistry[From any, To any, Key comparable] map[Key]func(From) (To, error)

func New[From any, To any, Key comparable]() Mapper[From, To, Key] {
	return make(mapRegistry[From, To, Key])
}

func (r mapRegistry[From, To, Key]) Register(key Key, converter func(From) (To, error)) {
	(r)[key] = converter
}

func (r mapRegistry[From, To, Key]) Convert(key Key, from From) (To, error) {
	converter, ok := (r)[key]
	if !ok {
		var zero To
		return zero, ErrUnknown
	}

	return converter(from)
}
