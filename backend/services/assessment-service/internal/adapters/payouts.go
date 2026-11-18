package adapters

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"errors"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	pb "saya/contracts/workflow/v1"
)

func (p *TestPayments) post(ctx context.Context, endpoint string, fields map[string]string, key string) (map[string]string, error) {
	script := path.Base(endpoint)
	fields["pg_sig"] = signature(script, fields, key)
	form := url.Values{}
	for k, v := range fields {
		form.Set(k, v)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", p.providerURL+endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := http.Client{Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.New("payment provider unavailable")
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 16384))
	if err != nil {
		return nil, err
	}
	data, err := flatXML(string(raw))
	if err != nil || resp.StatusCode != 200 || subtle.ConstantTimeCompare([]byte(data["pg_sig"]), []byte(signature(script, data, key))) != 1 {
		return nil, errors.New("unverified provider response")
	}
	return data, nil
}

// Claim before contacting Freedom Pay. After an interrupted submission only query its status.
func (p *TestPayments) ProcessPayout(ctx context.Context) error {
	var id, recipient, token, prev string
	var amount int
	err := p.repo.DB.QueryRowContext(ctx, `WITH candidate AS (
 SELECT id,payout_state FROM test_donations WHERE succeeded AND payout_state IN ('queued','checking') AND payout_due<=now()
 ORDER BY payout_due,id LIMIT 1 FOR UPDATE SKIP LOCKED)
 UPDATE test_donations d SET payout_state='checking',payout_due=now()+interval '5 minutes' FROM candidate c
 WHERE d.id=c.id RETURNING d.id,d.recipient_id,d.recipient_token,d.amount,c.payout_state`).Scan(&id, &recipient, &token, &amount, &prev)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	order := "p-" + id
	fields := map[string]string{"pg_merchant_id": p.merchant, "pg_order_id": order, "pg_salt": uuid.NewString(), "pg_testing_mode": "1"}
	endpoint := "/api/payment_status2"
	if prev == "queued" {
		endpoint = "/api/reg2reg"
		fields["pg_amount"] = strconv.Itoa(amount)
		fields["pg_currency"] = "KZT"
		fields["pg_user_id"] = recipient
		fields["pg_card_token_to"] = token
		fields["pg_description"] = "SAYA test payout"
		fields["pg_post_link"] = strings.TrimSuffix(p.callback, "result") + "payout"
	}
	data, err := p.post(ctx, endpoint, fields, p.payoutKey)
	if err != nil {
		return err
	}
	if data["pg_order_id"] != order || data["pg_merchant_id"] != p.merchant {
		return errors.New("unmatched payout result")
	}
	outcome := data["pg_payment_status"]
	amountValue := data["pg_amount"]
	if prev == "queued" {
		outcome = data["pg_status"]
		amountValue = data["pg_payment_amount"]
	}
	if outcome == "process" {
		return nil
	}
	result := "declined"
	switch outcome {
	case "ok", "success":
		amountText, _, _ := strings.Cut(amountValue, ".")
		n, err := strconv.Atoi(amountText)
		if err != nil || n != amount || !wholeAmount.MatchString(amountValue) || !paymentID.MatchString(data["pg_payment_id"]) || (data["pg_currency"] != "" && data["pg_currency"] != "KZT") {
			return errors.New("unmatched payout amount")
		}
		result = "paid"
	case "error":
		if prev != "queued" && data["pg_status"] != "ok" {
			return errors.New("payout status unavailable")
		}
	default:
		return errors.New("unknown payout status")
	}
	_, err = p.repo.DB.ExecContext(ctx, "UPDATE test_donations SET payout_state=$2,payout_id=NULLIF($3,'') WHERE id=$1 AND payout_state='checking'", id, result, data["pg_payment_id"])
	return err
}
func (s Server) PayoutCallback(ctx context.Context, in *pb.CardNotification) (*pb.Empty, error) {
	p := s.Payments
	if p == nil {
		return nil, status.Error(codes.Unavailable, "test payments unavailable")
	}
	f, err := flatXML(in.Xml)
	if err != nil || subtle.ConstantTimeCompare([]byte(f["pg_sig"]), []byte(signature("payout", f, p.payoutKey))) != 1 {
		return nil, status.Error(codes.PermissionDenied, "invalid notification")
	}
	id, ok := strings.CutPrefix(f["pg_order_id"], "p-")
	if !ok || uuid.Validate(id) != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid order")
	}
	_, err = p.repo.DB.ExecContext(ctx, "UPDATE test_donations SET payout_due=now() WHERE id=$1 AND payout_state='checking'", id)
	return &pb.Empty{}, appealError(err)
}
