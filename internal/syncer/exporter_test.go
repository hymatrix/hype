package syncer

import (
	"bytes"
	"testing"

	syncSchema "github.com/hymatrix/hype/internal/syncer/schema"
	goarSchema "github.com/permadao/goar/schema"
)

func TestExportLinesWithProgress_CallsCallback(t *testing.T) {
	db := newFakeDB()
	items := []syncSchema.ImportItem{
		{Nonce: 0, Msg: goarSchema.BundleItem{Id: "m0"}, Assign: goarSchema.BundleItem{Id: "a0"}},
		{Nonce: 1, Msg: goarSchema.BundleItem{Id: "m1"}, Assign: goarSchema.BundleItem{Id: "a1"}},
		{Nonce: 2, Msg: goarSchema.BundleItem{Id: "m2"}, Assign: goarSchema.BundleItem{Id: "a2"}},
	}
	if err := importItems(db, "p1", items, false); err != nil {
		t.Fatalf("ImportItems failed: %v", err)
	}

	var calls [][2]int64
	progress := func(done, total int64) {
		calls = append(calls, [2]int64{done, total})
	}

	var buf bytes.Buffer
	if err := exportLinesWithProgress(db, "p1", &buf, 2, progress); err != nil {
		t.Fatalf("ExportLinesWithProgress failed: %v", err)
	}
	if len(calls) == 0 {
		t.Fatalf("expected progress calls > 0")
	}
	last := calls[len(calls)-1]
	if last[0] != 3 || last[1] != 3 {
		t.Fatalf("expected final progress 3/3, got %d/%d", last[0], last[1])
	}
}
