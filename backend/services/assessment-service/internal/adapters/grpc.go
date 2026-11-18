package adapters

import (
	"context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"saya/assessment-service/internal/application"
	"saya/assessment-service/internal/domain"
	pb "saya/contracts/workflow/v1"
)

type Source struct{ Client pb.SurveyClient }

func (s Source) Fetch(c context.Context, id, user string) (domain.Signal, error) {
	v, e := s.Client.Input(c, &pb.OwnedID{Id: id, UserId: user})
	if e != nil {
		if status.Code(e) == codes.NotFound {
			return domain.Signal{}, domain.ErrNotFound
		}
		return domain.Signal{}, e
	}
	return domain.Signal{Ratings: v.Ratings, Summary: v.Summary, Needs: v.Needs, Type: "survey", SourceVersion: v.Version, ImmediateThreat: v.Answers["immediate_threat"], InformationRequested: v.Answers["information_requested"], ExternalConsent: v.ExternalLlmConsent}, nil
}

type Server struct {
	pb.UnimplementedAssessmentServer
	Service application.Service
}

func (s Server) Get(c context.Context, v *pb.OwnedID) (*pb.Result, error) {
	r, e := s.Service.Get(c, v.Id, v.UserId)
	if e != nil {
		if e == domain.ErrNotFound {
			return nil, status.Error(codes.NotFound, "assessment unavailable")
		}
		return nil, status.Error(codes.Internal, "assessment unavailable")
	}
	return &pb.Result{Version: r.Version, Ratings: r.Ratings, Percentage: r.Percentage, Summary: r.Summary, Directions: r.Directions, Id: r.ID, Status: r.Status, ImmediateThreatReported: r.ImmediateThreat, Recommendations: r.Recommendations, Disclaimer: r.Disclaimer, Provider: r.Provider}, nil
}
func (s Server) Input(c context.Context, v *pb.OwnedID) (*pb.Submission, error) {
	r, e := s.Service.Repo.Input(c, v.Id, v.UserId)
	if e != nil {
		return nil, status.Error(codes.NotFound, "input unavailable")
	}
	return &pb.Submission{UserId: v.UserId, Version: r.SourceVersion, Ratings: r.Ratings, Needs: r.Needs, ExternalLlmConsent: r.ExternalConsent, Answers: map[string]bool{"immediate_threat": r.ImmediateThreat, "information_requested": r.InformationRequested}}, nil
}
func (s Server) Complete(c context.Context, v *pb.Completion) (*pb.Empty, error) {
	e := s.Service.Complete(c, v.Id, v.UserId, v.Recommendations, v.Provider, v.Failed)
	if e == domain.ErrInvalid {
		return nil, status.Error(codes.InvalidArgument, "invalid structured result")
	}
	if e != nil {
		return nil, status.Error(codes.Internal, "completion failed")
	}
	return &pb.Empty{}, nil
}
