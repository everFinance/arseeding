package seeding

type Store struct {
	// TODO
}

func NewStore() *Store {
	// DB bot
	// TODO
	return &Store{}
}

func (s *Store) IsExist(arid string) bool {
	// TODO
	return false
}

func (s *Store) SaveTx(artx, offset interface{}) error {
	// TODO
	return nil
}

func (s *Store) LoadTx(arid string) (interface{}, error) {
	// TODO
	return nil, nil
}

func (s *Store) SaveChunk(offset, chunk interface{}) error {
	// TODO
	return nil
}

func (s *Store) LoadChunk(offset interface{}) (interface{}, error) {
	// TODO
	return nil, nil
}
