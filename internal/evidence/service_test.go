package evidence

import (
	"testing"
	"time"
)

func TestValidate(t *testing.T) {
	now := time.Now()
	later := now.Add(24 * time.Hour)
	if err := Validate(Item{Title: "A", Status: "active", Category: "Security", ReminderDaysBefore: 30, IssueDate: &now, ExpiryDate: &later}); err != nil {
		t.Fatal(err)
	}
	if err := Validate(Item{Status: "active", Category: "Security"}); err == nil {
		t.Fatal("expected error")
	}
	if err := Validate(Item{Title: "A", Status: "bad", Category: "Security"}); err == nil {
		t.Fatal("expected error")
	}
}
