package order

import (
	"reflect"
	"testing"
)

// Tabel transisi legal dari §11.4, ditulis ulang di test agar perubahan aturan
// yang tidak disengaja langsung ketahuan.
func TestCanTransitionMatchesSpecTable(t *testing.T) {
	legal := map[Status][]Status{
		StatusPendingPayment:  {StatusPaid, StatusCancelled, StatusExpired},
		StatusPaid:            {StatusProcessing, StatusRefundRequested, StatusCancelled},
		StatusProcessing:      {StatusShipped, StatusRefundRequested},
		StatusShipped:         {StatusDelivered},
		StatusDelivered:       {StatusCompleted, StatusRefundRequested},
		StatusRefundRequested: {StatusRefunded},
	}

	for from, targets := range legal {
		for _, to := range targets {
			if !CanTransition(from, to) {
				t.Errorf("CanTransition(%s, %s) = false, seharusnya true (§11.4)", from, to)
			}
		}
	}
}

func TestIllegalTransitionsAreRejected(t *testing.T) {
	illegal := [][2]Status{
		{StatusPendingPayment, StatusShipped},   // belum bayar tidak bisa langsung dikirim
		{StatusPendingPayment, StatusCompleted}, // harus lewat paid
		{StatusPaid, StatusDelivered},           // harus lewat processing & shipped
		{StatusProcessing, StatusDelivered},     // belum dikirim
		{StatusShipped, StatusCancelled},        // setelah dikirim tidak bisa dibatalkan
		{StatusCompleted, StatusRefunded},       // harus lewat refund_requested
		{StatusCancelled, StatusPaid},           // status final
		{StatusExpired, StatusPaid},             // status final
		{StatusRefunded, StatusPaid},            // status final
		{StatusDelivered, StatusShipped},        // tidak bisa mundur
	}

	for _, pair := range illegal {
		if CanTransition(pair[0], pair[1]) {
			t.Errorf("CanTransition(%s, %s) = true, seharusnya false (§11.4)", pair[0], pair[1])
		}
	}
}

func TestTransitionToSelfIsNeverAllowed(t *testing.T) {
	for _, s := range AllStatuses() {
		if CanTransition(s, s) {
			t.Errorf("CanTransition(%s, %s) harus false: transisi ke diri sendiri tidak bermakna", s, s)
		}
	}
}

func TestFinalStatusesHaveNoTransitions(t *testing.T) {
	for _, s := range []Status{StatusCompleted, StatusCancelled, StatusExpired, StatusRefunded} {
		if !s.IsFinal() {
			t.Errorf("%s harus IsFinal", s)
		}
		if got := AllowedTransitions(s); len(got) != 0 {
			t.Errorf("AllowedTransitions(%s) = %v, ingin kosong", s, got)
		}
	}
}

func TestNonFinalStatusesAreNotFinal(t *testing.T) {
	for _, s := range []Status{
		StatusPendingPayment, StatusPaid, StatusProcessing,
		StatusShipped, StatusDelivered, StatusRefundRequested,
	} {
		if s.IsFinal() {
			t.Errorf("%s tidak boleh IsFinal", s)
		}
	}
}

func TestAllStatusesAreValid(t *testing.T) {
	statuses := AllStatuses()
	if len(statuses) != 10 {
		t.Fatalf("AllStatuses() = %d status, ingin 10 sesuai enum order_status (§11.2)", len(statuses))
	}
	for _, s := range statuses {
		if !s.IsValid() {
			t.Errorf("%s harus IsValid", s)
		}
		if _, err := Parse(string(s)); err != nil {
			t.Errorf("Parse(%q) gagal: %v", s, err)
		}
	}
}

func TestUnknownStatusIsRejected(t *testing.T) {
	if Status("dikirim").IsValid() {
		t.Error("status di luar enum tidak boleh valid")
	}
	if _, err := Parse("dikirim"); err == nil {
		t.Error("Parse harus menolak status tak dikenal")
	}
	if _, err := Parse(""); err == nil {
		t.Error("Parse harus menolak string kosong")
	}
}

func TestParseIsCaseAndSpaceInsensitive(t *testing.T) {
	got, err := Parse("  PAID  ")
	if err != nil {
		t.Fatalf("Parse gagal: %v", err)
	}
	if got != StatusPaid {
		t.Errorf("Parse = %q, ingin %q", got, StatusPaid)
	}
}

func TestAllowedTransitionsIsSortedCopy(t *testing.T) {
	got := AllowedTransitions(StatusPendingPayment)
	want := []Status{StatusCancelled, StatusExpired, StatusPaid}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("AllowedTransitions = %v, ingin %v (terurut)", got, want)
	}

	// Mengubah hasil tidak boleh merusak tabel internal.
	got[0] = StatusCompleted
	again := AllowedTransitions(StatusPendingPayment)
	if again[0] == StatusCompleted {
		t.Error("AllowedTransitions mengembalikan slice internal, harus salinan")
	}
}

func TestRefundRejectionTargets(t *testing.T) {
	for _, s := range []Status{StatusPaid, StatusProcessing, StatusDelivered} {
		if !CanRejectRefund(s) {
			t.Errorf("CanRejectRefund(%s) harus true", s)
		}
	}
	for _, s := range []Status{StatusPendingPayment, StatusShipped, StatusCompleted, StatusCancelled} {
		if CanRejectRefund(s) {
			t.Errorf("CanRejectRefund(%s) harus false", s)
		}
	}
}

func TestRefundableStatuses(t *testing.T) {
	for _, s := range []Status{StatusPaid, StatusProcessing, StatusDelivered} {
		if !s.Refundable() {
			t.Errorf("%s harus Refundable", s)
		}
	}
	if StatusPendingPayment.Refundable() {
		t.Error("pending_payment belum dibayar, tidak bisa refund")
	}
}

func TestCancellableByCustomer(t *testing.T) {
	if !StatusPendingPayment.CancellableByCustomer() {
		t.Error("pembeli boleh membatalkan sebelum bayar")
	}
	if !StatusPaid.CancellableByCustomer() {
		t.Error("pembeli boleh membatalkan setelah bayar sebelum dikemas")
	}
	if StatusShipped.CancellableByCustomer() {
		t.Error("setelah dikirim, pembatalan harus lewat admin (§8.6 ORD_CANCEL_DENIED)")
	}
	if StatusDelivered.CancellableByCustomer() {
		t.Error("setelah diterima, pembatalan harus lewat refund")
	}
}

func TestPaidFamily(t *testing.T) {
	paidFamily := []Status{StatusPaid, StatusProcessing, StatusShipped, StatusDelivered, StatusCompleted}
	for _, s := range paidFamily {
		if !s.IsPaidFamily() {
			t.Errorf("%s harus IsPaidFamily", s)
		}
	}
	for _, s := range []Status{StatusPendingPayment, StatusCancelled, StatusExpired, StatusRefundRequested, StatusRefunded} {
		if s.IsPaidFamily() {
			t.Errorf("%s tidak boleh IsPaidFamily", s)
		}
	}
}
