package adapters

import (
	"context"
	"crypto/subtle"
	"encoding/xml"
	"errors"

	"net/url"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	pb "saya/contracts/workflow/v1"
)

func flatXML(raw string) (map[string]string, error) {
	var msg struct {
		XMLName xml.Name `xml:"response"`
		Fields  []struct {
			XMLName  xml.Name
			Value    string     `xml:",chardata"`
			Children []struct{} `xml:",any"`
		} `xml:",any"`
	}
	if err := xml.Unmarshal([]byte(raw), &msg); err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, f := range msg.Fields {
		key := f.XMLName.Local
		if _, exists := out[key]; exists || len(f.Children) > 0 {
			return nil, errors.New("invalid fields")
		}
		out[key] = f.Value
	}
	return out, nil
}
func (s Server) BindPayoutCard(ctx context.Context, in *pb.OwnedID) (*pb.CheckoutReply, error) {
	p := s.Payments
	if p == nil {
		return nil, status.Error(codes.Unavailable, "test payments unavailable")
	}
	if _, err := uuid.Parse(in.Id); err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid appeal")
	}
	var binding, order string
	err := p.repo.DB.QueryRowContext(ctx, `INSERT INTO payout_bindings(user_id,id,pending_id)
 SELECT user_id,$3,$4 FROM eligible_appeals WHERE user_id=$1 AND id=$2 AND state IN ('published','pending')
 ON CONFLICT(user_id) DO UPDATE SET pending_id=excluded.pending_id RETURNING id,pending_id`, in.UserId, in.Id, uuid.NewString(), uuid.NewString()).Scan(&binding, &order)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, "eligible appeal required")
	}
	fields := map[string]string{"pg_merchant_id": p.merchant, "pg_order_id": order, "pg_user_id": binding, "pg_post_link": strings.TrimSuffix(p.callback, "result") + "card", "pg_salt": uuid.NewString(), "pg_testing_mode": "1"}
	data, err := p.post(ctx, "/v1/merchant/"+url.PathEscape(p.merchant)+"/cardstoragepayout/add", fields, p.key)
	if err != nil || data["pg_status"] != "ok" {
		return nil, status.Error(codes.Unavailable, "card service unavailable")
	}
	u, err := url.Parse(data["pg_redirect_url"])
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil {
		return nil, status.Error(codes.Unavailable, "invalid provider URL")
	}
	return &pb.CheckoutReply{Url: u.String()}, nil
}
func (s Server) CardCallback(ctx context.Context, in *pb.CardNotification) (*pb.Empty, error) {
	p := s.Payments
	if p == nil {
		return nil, status.Error(codes.Unavailable, "test payments unavailable")
	}
	f, err := flatXML(in.Xml)
	if err != nil || subtle.ConstantTimeCompare([]byte(f["pg_sig"]), []byte(signature("card", f, p.key))) != 1 {
		return nil, status.Error(codes.PermissionDenied, "invalid card notification")
	}
	id, err := uuid.Parse(f["pg_order_id"])
	if err != nil || uuid.Validate(f["pg_user_id"]) != nil || f["pg_type"] != "add_payout_card" {
		return nil, status.Error(codes.InvalidArgument, "invalid binding")
	}
	if f["pg_status"] == "error" {
		return &pb.Empty{}, nil
	}
	token := f["pg_card_token"]
	if f["pg_status"] != "success" || len(token) == 0 || len(token) > 256 {
		return nil, status.Error(codes.InvalidArgument, "invalid binding")
	}
	err = applied(p.repo.DB.ExecContext(ctx, "UPDATE payout_bindings SET card_token=$2 WHERE pending_id=$1 AND id=$3", id, token, f["pg_user_id"]))
	return &pb.Empty{}, appealError(err)
}
