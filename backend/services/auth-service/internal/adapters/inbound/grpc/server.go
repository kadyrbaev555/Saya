package grpcadapter

import (
	"context"
	"errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"saya/auth-service/internal/application"
	"saya/auth-service/internal/domain"
	pb "saya/contracts/auth/v1"
)

type Server struct {
	pb.UnimplementedAuthServer
	Service        *application.Service
	RealtimeSigner application.RealtimeSigner
}

func mapped(e error) error {
	switch {
	case e == nil:
		return nil
	case errors.Is(e, domain.ErrValidation):
		return status.Error(codes.InvalidArgument, "invalid input")
	case errors.Is(e, domain.ErrConflict):
		return status.Error(codes.AlreadyExists, "account unavailable")
	case errors.Is(e, domain.ErrCredentials):
		return status.Error(codes.Unauthenticated, "invalid credentials")
	default:
		return status.Error(codes.Internal, "operation failed")
	}
}
func user(u domain.User) *pb.User { return &pb.User{Id: u.ID, Email: u.Email, Role: u.Role} }
func session(s domain.Session, e error) (*pb.Session, error) {
	if e != nil {
		return nil, mapped(e)
	}
	return &pb.Session{AccessToken: s.Access, RefreshToken: s.Refresh, User: user(s.User)}, nil
}
func (s *Server) Register(c context.Context, r *pb.Credentials) (*pb.Session, error) {
	v, e := s.Service.Register(c, r.Email, r.Password)
	return session(v, e)
}
func (s *Server) Login(c context.Context, r *pb.Credentials) (*pb.Session, error) {
	v, e := s.Service.Login(c, r.Email, r.Password)
	return session(v, e)
}
func (s *Server) Refresh(c context.Context, r *pb.RefreshRequest) (*pb.Session, error) {
	v, e := s.Service.Refresh(c, r.Token)
	return session(v, e)
}
func (s *Server) Logout(c context.Context, r *pb.RefreshRequest) (*pb.Empty, error) {
	e := s.Service.Logout(c, r.Token)
	return &pb.Empty{}, mapped(e)
}
func (s *Server) Me(c context.Context, r *pb.Identity) (*pb.User, error) {
	u, e := s.Service.Repo.Get(c, r.UserId)
	if e != nil {
		return nil, status.Error(codes.NotFound, "user unavailable")
	}
	return user(u), nil
}

func (s *Server) Realtime(c context.Context, r *pb.RealtimeRequest) (*pb.RealtimeToken, error) {
	v, exp, e := s.Service.Realtime(c, r.UserId, r.Channel, s.RealtimeSigner)
	if e != nil {
		return nil, mapped(e)
	}
	return &pb.RealtimeToken{Token: v, ExpiresAt: exp, Channel: r.Channel}, nil
}
