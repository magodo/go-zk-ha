package zkha

type Service struct {
	opts Options
}

func NewService(opts ...Option) (*Service, error) {
	s := &Service{}
	s.opts = defaultOptions()
	for _, o := range opts {
		o(&s.opts)
	}

	if err := s.opts.Err(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Service) Run() error {
	sm, err := initSm(s)
	if err != nil {
		return err
	}

	for {
		sm.transit()
	}
}
