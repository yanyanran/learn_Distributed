package service

import "context"

type Service interface {
	TestAdd(ctx context.Context, in Add) AddAck
}

type baseServer struct {
}

