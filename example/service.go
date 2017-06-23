package example

import "context"

type service struct{}

func (s *service) Sum(ctx context.Context, req *Req) (*Rep, error) {
	return &Rep{
		Sum: req.Left + req.Right,
	}, nil
}

func NewService() Service {
	return &service{}
}
