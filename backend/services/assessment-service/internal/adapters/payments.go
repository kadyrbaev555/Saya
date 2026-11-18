package adapters

import (
	"context"
	"crypto/md5" // Freedom Pay's signature format.
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"net/url"

	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	pb "saya/contracts/workflow/v1"
)

type TestPayments struct {
	repo                               Repository
	merchant, key, payoutKey, callback string
	providerURL                        string
}

func NewTestPayments(repo Repository, merchant, key, payoutKey, callback string) (*TestPayments, error) {
	p := &TestPayments{repo: repo, providerURL: "https://api.freedompay.kz", payoutKey: payoutKey, merchant: merchant, key: key, callback: callback}
	if p.merchant == "" && p.key == "" && p.callback == "" && p.payoutKey == "" {
		return nil, nil
	}
	u, err := url.Parse(p.callback)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.Path != "/api/v1/payments/result" || u.RawQuery != "" || u.Fragment != "" || p.merchant == "" || p.key == "" || p.payoutKey == "" {
		return nil, errors.New("invalid Freedom Pay test configuration")
	}
	return p, nil
}
func signature(script string, fields map[string]string, key string) string {
	names := make([]string, 0, len(fields))
	for k := range fields {
		if k != "pg_sig" {
			names = append(names, k)
		}
	}
	sort.Strings(names)
	parts := []string{script}
	for _, k := range names {
		parts = append(parts, fields[k])
	}
	parts = append(parts, key)
	digest := md5.Sum([]byte(strings.Join(parts, ";")))
	return hex.EncodeToString(digest[:])
}
func (s Server) Checkout(ctx context.Context, in *pb.CheckoutRequest) (*pb.CheckoutReply, error) {
	p := s.Payments
	if p == nil {
		return nil, status.Error(codes.Unavailable, "test checkout unavailable")
	}
	if _, err := uuid.Parse(in.AppealId); err != nil || in.Amount < 100 || in.Amount > 100000 {
		return nil, status.Error(codes.InvalidArgument, "invalid checkout")
	}
	order := uuid.NewString()
	err := applied(p.repo.DB.ExecContext(ctx, `INSERT INTO test_donations(id,user_id,appeal_id,amount,recipient_id,recipient_token) SELECT $1,a.user_id,a.id,$3,b.id,b.card_token FROM eligible_appeals a JOIN payout_bindings b ON b.user_id=a.user_id WHERE a.id=$2 AND a.state='published' AND b.card_token IS NOT NULL`, order, in.AppealId, in.Amount))
	if err != nil {
		return nil, appealError(err)
	}
	fields := map[string]string{"pg_merchant_id": p.merchant, "pg_order_id": order, "pg_amount": strconv.Itoa(int(in.Amount)), "pg_currency": "KZT", "pg_description": "SAYA test donation", "pg_salt": uuid.NewString(), "pg_result_url": p.callback, "pg_request_method": "POST", "pg_auto_clearing": "1", "pg_testing_mode": "1"}
	fields["pg_sig"] = signature("payment.php", fields, p.key)
	q := url.Values{}
	for k, v := range fields {
		q.Set(k, v)
	}
	return &pb.CheckoutReply{Url: "https://api.freedompay.kz/payment.php?" + q.Encode()}, nil
}

var wholeAmount = regexp.MustCompile(`^[0-9]{1,6}(\.0{1,2})?$`)
var paymentID = regexp.MustCompile(`^[0-9]{1,32}$`)

func (s Server) PaymentCallback(ctx context.Context, in *pb.PaymentResult) (*pb.PaymentAck, error) {
	p := s.Payments
	if p == nil {
		return nil, status.Error(codes.Unavailable, "test checkout unavailable")
	}
	f := in.Fields
	if subtle.ConstantTimeCompare([]byte(f["pg_sig"]), []byte(signature("result", f, p.key))) != 1 {
		return nil, status.Error(codes.PermissionDenied, "invalid signature")
	}
	_, err := uuid.Parse(f["pg_order_id"])
	result := f["pg_result"]
	if err != nil || !paymentID.MatchString(f["pg_payment_id"]) || !wholeAmount.MatchString(f["pg_amount"]) || f["pg_currency"] != "KZT" || (result != "0" && result != "1") || (result == "1" && f["pg_captured"] != "1") {
		return nil, status.Error(codes.InvalidArgument, "invalid payment result")
	}
	amount, _, _ := strings.Cut(f["pg_amount"], ".")
	err = applied(p.repo.DB.ExecContext(ctx, `UPDATE test_donations SET provider_id=$2,succeeded=$3 WHERE id=$1 AND amount=$4 AND (provider_id IS NULL OR (provider_id=$2 AND succeeded=$3))`, f["pg_order_id"], f["pg_payment_id"], result == "1", amount))
	if err != nil {
		return nil, appealError(err)
	}
	return &pb.PaymentAck{Xml: "<response><pg_status>ok</pg_status></response>"}, nil
}
