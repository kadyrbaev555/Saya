package adapters

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"net/http"
	"net/http/httptest"
	"net/url"
	"saya/assessment-service/internal/application"
	"saya/assessment-service/internal/domain"
	pb "saya/contracts/workflow/v1"
	"strings"
	"testing"
)

func TestPaymentSignature(t *testing.T) {
	values := map[string]string{"pg_amount": "100", "pg_currency": "KZT", "pg_description": "Order #1001", "pg_merchant_id": "12345", "pg_order_id": "order_1001", "pg_salt": "a1b2c3d4"}
	if signature("init_payment.php", values, "test_secret_key") != "b9ad75871668ca040f80f5f6b530e6ed" {
		t.Fatal("provider signature mismatch")
	}
}
func checkAppeals(t *testing.T, r Repository) {
	ctx := context.Background()
	user, id := uuid.NewString(), uuid.NewString()
	signal := domain.Signal{SourceVersion: 2, Needs: []string{"financial"}, Ratings: map[string]int32{"expenses": 0}}
	if err := r.Create(ctx, id, user, signal); err != nil {
		t.Fatal(err)
	}
	if err := r.Complete(ctx, id, user, nil, "self-report", false); err != nil {
		t.Fatal(err)
	}
	service := application.AppealService{Store: r}
	if err := service.Submit(ctx, user, id, "public", false); !errors.Is(err, application.ErrInvalidAppeal) {
		t.Fatal("missing consent", err)
	}
	if err := service.Submit(ctx, uuid.NewString(), id, "public", true); !errors.Is(err, application.ErrAppealUnavailable) {
		t.Fatal("foreign assessment", err)
	}
	if err := service.Submit(ctx, user, id, "public", true); err != nil {
		t.Fatal(err)
	}
	list := func(scope string) []application.Appeal {
		t.Helper()
		items, err := r.ListAppeals(ctx, scope, user)
		if err != nil {
			t.Fatal(err)
		}
		return items
	}
	if len(list("public")) != 0 {
		t.Fatal("pending published")
	}
	item := list("mine")[0]
	if err := r.ChangeAppeal(ctx, "", item.ID, "approve"); err != nil {
		t.Fatal(err)
	}
	if len(list("public")) != 1 {
		t.Fatal("missing publication")
	}
	if err := r.ChangeAppeal(ctx, "", item.ID, "report"); err != nil || !list("review")[0].Reported {
		t.Fatal("report", err)
	}
	server := Server{Payments: &TestPayments{repo: r, merchant: "demo", callback: "https://example.test/api/v1/payments/result", key: "test"}}
	if _, err := server.Checkout(ctx, &pb.CheckoutRequest{AppealId: item.ID, Amount: 500}); err == nil {
		t.Fatal("unbound recipient")
	}
	binding, order := uuid.NewString(), uuid.NewString()
	if _, err := r.DB.Exec("INSERT INTO payout_bindings(id,user_id,pending_id) VALUES ($1,$2,$3)", binding, user, order); err != nil {
		t.Fatal(err)
	}
	card := map[string]string{"pg_status": "success", "pg_user_id": binding, "pg_order_id": order, "pg_type": "add_payout_card", "pg_card_token": "test-token"}
	card["pg_sig"] = signature("card", card, "test")
	raw := "<response>"
	for k, v := range card {
		raw += "<" + k + ">" + v + "</" + k + ">"
	}
	raw += "</response>"
	if _, err := server.CardCallback(ctx, &pb.CardNotification{Xml: raw}); err != nil {
		t.Fatal(err)
	}
	checkout, err := server.Checkout(ctx, &pb.CheckoutRequest{AppealId: item.ID, Amount: 500})
	if err != nil {
		t.Fatal(err)
	}
	redirect, _ := url.Parse(checkout.Url)
	q := redirect.Query()
	if q.Get("pg_testing_mode") != "1" {
		t.Fatal("unsafe checkout")
	}
	fields := map[string]string{"pg_order_id": q.Get("pg_order_id"), "pg_payment_id": "123", "pg_amount": "500.00", "pg_currency": "KZT", "pg_result": "1", "pg_captured": "1"}
	cb := func() error { _, err := server.PaymentCallback(ctx, &pb.PaymentResult{Fields: fields}); return err }
	if cb() == nil {
		t.Fatal("unsigned callback accepted")
	}
	fields["pg_sig"] = signature("result", fields, "test")
	for range 2 {
		if err := cb(); err != nil {
			t.Fatal(err)
		}
	}
	var sum int
	if err := r.DB.Get(&sum, "SELECT sum(amount) FROM test_donations WHERE succeeded"); err != nil || sum != 500 {
		t.Fatal("duplicate", err, sum)
	}
	if err := r.ChangeAppeal(ctx, uuid.NewString(), item.ID, "withdraw"); !errors.Is(err, application.ErrAppealUnavailable) {
		t.Fatal("foreign withdrawal", err)
	}
	// A lost response must lead to status reconciliation, never to a second payout.
	var paths []string
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		paths = append(paths, req.URL.Path)
		req.ParseForm()
		fields := map[string]string{}
		for k := range req.PostForm {
			fields[k] = req.PostForm.Get(k)
		}
		script := strings.TrimPrefix(req.URL.Path, "/api/")
		if fields["pg_sig"] != signature(script, fields, "out") || fields["pg_testing_mode"] != "1" {
			t.Error("bad payout request")
		}
		if script == "reg2reg" {
			if fields["pg_card_token_to"] != "test-token" {
				t.Error("wrong recipient")
			}
			w.WriteHeader(503)
			return
		}
		response := map[string]string{"pg_order_id": fields["pg_order_id"], "pg_merchant_id": "demo", "pg_status": "ok", "pg_currency": "KZT", "pg_payment_status": "success", "pg_amount": "500.00", "pg_payment_id": "999"}
		response["pg_sig"] = signature(script, response, "out")
		fmt.Fprint(w, "<response>")
		for k, v := range response {
			fmt.Fprintf(w, "<%s>", k)
			xml.EscapeText(w, []byte(v))
			fmt.Fprintf(w, "</%s>", k)
		}
		fmt.Fprint(w, "</response>")
	}))
	defer provider.Close()
	server.Payments.providerURL = provider.URL
	server.Payments.payoutKey = "out"
	if err := server.Payments.ProcessPayout(ctx); err == nil {
		t.Fatal("unexpected provider success")
	}
	if _, err := r.DB.Exec("UPDATE test_donations SET payout_due=now()"); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := server.Payments.ProcessPayout(ctx); err != nil {
			t.Fatal(err)
		}
	}
	var payout string
	if err := r.DB.Get(&payout, "SELECT payout_id FROM test_donations WHERE payout_state='paid'"); err != nil || payout != "999" || strings.Join(paths, ",") != "/api/reg2reg,/api/payment_status2" {
		t.Fatal("payout reconciliation", err, paths)
	}
	next := uuid.NewString()
	signal.Needs = nil
	if err := r.Create(ctx, next, user, signal); err != nil {
		t.Fatal(err)
	}
	if err := r.Complete(ctx, next, user, nil, "self-report", false); err != nil {
		t.Fatal(err)
	}
	if len(list("public")) != 0 {
		t.Fatal("stale recommendation exposed")
	}
	if err := service.Submit(ctx, user, id, "stale", true); !errors.Is(err, application.ErrAppealUnavailable) {
		t.Fatal("stale submit", err)
	}
	if err := r.ChangeAppeal(ctx, user, item.ID, "withdraw"); err != nil {
		t.Fatal(err)
	}
}
