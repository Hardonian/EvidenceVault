package evidence

import (
	"testing"
	"time"
)

func TestStarterTemplatesAndLookup(t *testing.T) {
	ts := StarterTemplates(time.Now().UTC())
	if len(ts) != 6 {
		t.Fatalf("expected 6 templates")
	}
	if _, ok := TemplateByKey(time.Now().UTC(), "privacy-policy"); !ok {
		t.Fatal("expected template")
	}
}
