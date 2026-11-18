package adapters

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"saya/assessment-service/internal/application"
	pb "saya/contracts/workflow/v1"
)

func (r Repository) ListAppeals(ctx context.Context, scope, user string) ([]application.Appeal, error) {
	query := "SELECT id,description,state,reported,EXISTS(SELECT 1 FROM payout_bindings b WHERE b.user_id=a.user_id AND b.card_token IS NOT NULL) FROM eligible_appeals a WHERE state='published' ORDER BY id"
	var args []any
	switch scope {
	case "review":
		query = "SELECT id,description,state,reported,EXISTS(SELECT 1 FROM payout_bindings b WHERE b.user_id=a.user_id AND b.card_token IS NOT NULL) FROM eligible_appeals a WHERE state='pending' OR (state='published' AND reported) ORDER BY reported DESC,id"
	case "mine":
		query = `SELECT id,description,CASE WHEN state IN ('pending','published') AND NOT EXISTS
 (SELECT 1 FROM eligible_appeals e WHERE e.id=a.id) THEN 'expired' ELSE state END,
 reported,EXISTS(SELECT 1 FROM payout_bindings b WHERE b.user_id=a.user_id AND b.card_token IS NOT NULL) FROM appeals a WHERE user_id=$1`
		args = []any{user}
	case "public":
	default:
		return nil, application.ErrInvalidAppeal
	}
	rows, err := r.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []application.Appeal
	for rows.Next() {
		var a application.Appeal
		if err := rows.Scan(&a.ID, &a.Description, &a.State, &a.Reported, &a.CardReady); err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, rows.Err()
}
func applied(result sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err == nil && n == 0 {
		return application.ErrAppealUnavailable
	}
	return err
}
func (r Repository) SubmitAppeal(ctx context.Context, user, id, description string) error {
	return applied(r.DB.ExecContext(ctx, `INSERT INTO appeals(user_id,id,assessment_id,description,state)
 SELECT $1,$4,id,$3,'pending' FROM
 (SELECT id,result FROM assessments WHERE user_id=$1 AND completed_at IS NOT NULL ORDER BY completed_at DESC,id DESC LIMIT 1) latest
 WHERE id=$2 AND result->'directions' ? 'financial'
 ON CONFLICT(user_id) DO UPDATE SET id=excluded.id,assessment_id=excluded.assessment_id,description=excluded.description,state='pending',reported=false`, user, id, description, uuid.NewString()))
}
func (r Repository) ChangeAppeal(ctx context.Context, user, id, action string) error {
	switch action {
	case "withdraw":
		return applied(r.DB.ExecContext(ctx, "UPDATE appeals SET state='withdrawn' WHERE id=$1 AND user_id=$2", id, user))
	case "report":
		return applied(r.DB.ExecContext(ctx, "UPDATE appeals SET reported=true WHERE id=$1 AND id IN (SELECT id FROM eligible_appeals WHERE state='published')", id))
	case "approve", "decline":
		state := "declined"
		if action == "approve" {
			state = "published"
		}
		return applied(r.DB.ExecContext(ctx, "UPDATE appeals SET state=$2,reported=false WHERE id=$1 AND (state='pending' OR (state='published' AND reported)) AND id IN (SELECT id FROM eligible_appeals)", id, state))
	default:
		return application.ErrInvalidAppeal
	}
}
func appealError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, application.ErrInvalidAppeal):
		return status.Error(codes.InvalidArgument, "invalid appeal")
	case errors.Is(err, application.ErrAppealUnavailable):
		return status.Error(codes.NotFound, "appeal unavailable")
	default:
		return status.Error(codes.Internal, "appeals unavailable")
	}
}
func (s Server) Appeals(ctx context.Context, in *pb.AppealQuery) (*pb.AppealList, error) {
	if in.Scope == "review" && !in.Moderator {
		return nil, status.Error(codes.PermissionDenied, "restricted")
	}
	items, err := s.AppealService.Store.ListAppeals(ctx, in.Scope, in.UserId)
	if err != nil {
		return nil, appealError(err)
	}
	out := &pb.AppealList{CheckoutAvailable: s.Payments != nil}
	for _, a := range items {
		item := &pb.AppealItem{Id: a.ID, Description: a.Description, CardReady: a.CardReady}
		if in.Scope != "public" {
			item.State = a.State
			item.Reported = a.Reported
		}
		out.Items = append(out.Items, item)
	}
	return out, nil
}
func (s Server) MutateAppeal(ctx context.Context, in *pb.AppealChange) (*pb.Empty, error) {
	if (in.Action == "approve" || in.Action == "decline") && !in.Moderator {
		return nil, status.Error(codes.PermissionDenied, "restricted")
	}
	if _, err := uuid.Parse(in.Id); err != nil {
		return nil, appealError(application.ErrInvalidAppeal)
	}
	if in.Action == "submit" {
		return &pb.Empty{}, appealError(s.AppealService.Submit(ctx, in.UserId, in.Id, in.Description, in.Consent))
	}
	return &pb.Empty{}, appealError(s.AppealService.Store.ChangeAppeal(ctx, in.UserId, in.Id, in.Action))
}
