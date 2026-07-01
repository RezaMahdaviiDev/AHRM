package scanner

import (
	"testing"

	"ahrm/internal/sourcearena"
)

func TestClassifyHaltReason(t *testing.T) {
	cases := []struct {
		name   string
		reason string
		want   string
	}{
		{name: "assembly", reason: "به علت برگزاری مجمع عمومی سالیانه", want: haltReasonCategoryAssembly},
		{name: "disclosure", reason: "به علت افشای اطلاعات با اهمیت گروه الف", want: haltReasonCategoryDisclosure},
		{name: "volatility", reason: "به علت نوسان قیمت و ورود به حراج", want: haltReasonCategoryVolatility},
		{name: "clarify", reason: "به منظور شفاف سازی و رفع ابهام", want: haltReasonCategoryClarify},
		{name: "other", reason: "توقف موقت", want: defaultHaltReasonCategory},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifyHaltReason(tc.reason); got != tc.want {
				t.Fatalf("classifyHaltReason(%q)=%q want=%q", tc.reason, got, tc.want)
			}
		})
	}
}

func TestIsReopenSupervisorMessage(t *testing.T) {
	if !isReopenSupervisorMessage("اطلاعیه ناظر: آغاز بازگشایی نماد در مرحله پیش گشایش") {
		t.Fatal("expected reopen detection")
	}
	if isReopenSupervisorMessage("نماد تا اطلاع ثانوی متوقف است") {
		t.Fatal("halt message must not be treated as reopen")
	}
}

func TestLatestMessagesBySymbol(t *testing.T) {
	messages := []sourcearena.SupervisorMessage{
		{Symbol: "خساپا", Message: "آغاز بازگشایی"},
		{Symbol: "خساپا", Message: "آغاز بازگشایی قدیمی"},
		{Symbol: "فولاد", Message: "توقف نماد"},
	}
	got := latestMessagesBySymbol(messages, isReopenSupervisorMessage)
	if len(got) != 1 {
		t.Fatalf("len=%d", len(got))
	}
	if got["خساپا"].Message != "آغاز بازگشایی" {
		t.Fatalf("message=%q", got["خساپا"].Message)
	}
}
