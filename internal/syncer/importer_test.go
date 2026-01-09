package syncer

import (
	"bufio"
	"encoding/json"
	"os"
	"testing"

	nodeSchema "github.com/hymatrix/hymx/node/schema"
	vmmSchema "github.com/hymatrix/hymx/vmm/schema"
	syncSchema "github.com/hymatrix/hype/internal/syncer/schema"
	goarSchema "github.com/permadao/goar/schema"
)

type fakeDB struct {
	msgIndex    map[string]*goarSchema.BundleItem
	msgLists    map[string][]goarSchema.BundleItem
	assignLists map[string][]goarSchema.BundleItem
	nonceByPid  map[string]int64
}

func newFakeDB() *fakeDB {
	return &fakeDB{
		msgIndex:    make(map[string]*goarSchema.BundleItem),
		msgLists:    make(map[string][]goarSchema.BundleItem),
		assignLists: make(map[string][]goarSchema.BundleItem),
		nonceByPid:  make(map[string]int64),
	}
}

func (f *fakeDB) SaveResult(result vmmSchema.VmmResult) error             { return nil }
func (f *fakeDB) GetResult(string) (*vmmSchema.VmmResult, error)          { return nil, nil }
func (f *fakeDB) GetResults(string, int64) ([]vmmSchema.VmmResult, error) { return nil, nil }
func (f *fakeDB) IsExist(pid string) (bool, error) {
	_, ok := f.nonceByPid[pid]
	return ok, nil
}
func (f *fakeDB) GetNonce(pid string) (int64, error) {
	nonce, ok := f.nonceByPid[pid]
	if !ok {
		return -1, nil
	}
	return nonce, nil
}
func (f *fakeDB) Commit(pid string, nonce int64, msg, assign goarSchema.BundleItem) error {
	f.msgLists[pid] = append(f.msgLists[pid], msg)
	f.assignLists[pid] = append(f.assignLists[pid], assign)
	f.msgIndex[msg.Id] = &msg
	f.nonceByPid[pid] = nonce
	return nil
}
func (f *fakeDB) GetAllProcess() ([]string, []int64, error) { return nil, nil, nil }
func (f *fakeDB) GetMessage(msgid string) (*goarSchema.BundleItem, error) {
	return f.msgIndex[msgid], nil
}
func (f *fakeDB) GetMessageByNonce(pid string, nonce int64) (*goarSchema.BundleItem, error) {
	msgs := f.msgLists[pid]
	if nonce < 0 || int(nonce) >= len(msgs) {
		return nil, nil
	}
	m := msgs[nonce]
	return &m, nil
}
func (f *fakeDB) GetAssignByNonce(pid string, nonce int64) (*goarSchema.BundleItem, error) {
	assigns := f.assignLists[pid]
	if nonce < 0 || int(nonce) >= len(assigns) {
		return nil, nil
	}
	a := assigns[nonce]
	return &a, nil
}
func (f *fakeDB) GetCheckpointIndex(string) (string, error) { return "", nil }
func (f *fakeDB) SaveCheckpointIndex(string, string) error  { return nil }
func (f *fakeDB) GetCache(string, string) (string, error)   { return "", nil }
func (f *fakeDB) SaveCache(string, string, string) error    { return nil }

var _ nodeSchema.IDB = (*fakeDB)(nil)

func TestimportItems_SuccessAndForce(t *testing.T) {
	db := newFakeDB()
	items := []syncSchema.ImportItem{
		{Nonce: 0, Msg: goarSchema.BundleItem{Id: "msgid-1"}, Assign: goarSchema.BundleItem{Id: "assignid-1"}},
		{Nonce: 1, Msg: goarSchema.BundleItem{Id: "msgid-2"}, Assign: goarSchema.BundleItem{Id: "assignid-2"}},
	}
	sorted, err := checkAndSort(items)
	if err != nil {
		t.Fatalf("checkAndSort failed: %v", err)
	}
	if err := importItems(db, "p1", sorted, false); err != nil {
		t.Fatalf("importItems failed: %v", err)
	}
	if len(db.msgLists["p1"]) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(db.msgLists["p1"]))
	}
	// duplicate without force: skip
	if err := importItems(db, "p1", sorted, false); err != nil {
		t.Fatalf("importItems failed: %v", err)
	}
	if len(db.msgLists["p1"]) != 2 {
		t.Fatalf("expected 2 messages after skip, got %d", len(db.msgLists["p1"]))
	}
	// with force: append duplicates
	if err := importItems(db, "p1", sorted, true); err != nil {
		t.Fatalf("importItems failed: %v", err)
	}
	if len(db.msgLists["p1"]) != 4 {
		t.Fatalf("expected 4 messages after force, got %d", len(db.msgLists["p1"]))
	}
}

func TestCheckAndSort_Validations(t *testing.T) {
	// gap
	itemsGap := []syncSchema.ImportItem{
		{Nonce: 0, Msg: goarSchema.BundleItem{Id: "ga"}, Assign: goarSchema.BundleItem{Id: "ga"}},
		{Nonce: 2, Msg: goarSchema.BundleItem{Id: "gb"}, Assign: goarSchema.BundleItem{Id: "gb"}},
	}
	if _, err := checkAndSort(itemsGap); err == nil {
		t.Fatalf("expected error for gap, got nil")
	}
	// start not zero
	itemsStart1 := []syncSchema.ImportItem{
		{Nonce: 1, Msg: goarSchema.BundleItem{Id: "sa"}, Assign: goarSchema.BundleItem{Id: "sa"}},
	}
	if _, err := checkAndSort(itemsStart1); err == nil {
		t.Fatalf("expected error for start at 1, got nil")
	}
	// unsorted valid
	itemsUnsorted := []syncSchema.ImportItem{
		{Nonce: 2, Msg: goarSchema.BundleItem{Id: "u2"}, Assign: goarSchema.BundleItem{Id: "u2"}},
		{Nonce: 0, Msg: goarSchema.BundleItem{Id: "u0"}, Assign: goarSchema.BundleItem{Id: "u0"}},
		{Nonce: 1, Msg: goarSchema.BundleItem{Id: "u1"}, Assign: goarSchema.BundleItem{Id: "u1"}},
	}
	if _, err := checkAndSort(itemsUnsorted); err != nil {
		t.Fatalf("unexpected error for valid unsorted: %v", err)
	}
}

func TestJSONLExample_ParseAndImport(t *testing.T) {
	db := newFakeDB()
	f, err := os.Open("schema/example.jsonl")
	if err != nil {
		t.Fatalf("open jsonl failed: %v", err)
	}
	defer f.Close()
	var lines []syncSchema.ImportLine
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var ln syncSchema.ImportLine
		if err := json.Unmarshal([]byte(sc.Text()), &ln); err != nil {
			t.Fatalf("unmarshal line failed: %v", err)
		}
		lines = append(lines, ln)
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if len(lines) == 0 {
		t.Fatalf("expected lines > 0")
	}
	pid := lines[0].Pid
	items := make([]syncSchema.ImportItem, 0, len(lines))
	for _, ln := range lines {
		if ln.Pid != pid {
			t.Fatalf("pid mismatch: %s vs %s", ln.Pid, pid)
		}
		items = append(items, syncSchema.ImportItem{Nonce: ln.Nonce, Msg: ln.Msg, Assign: ln.Assign})
	}
	sorted, err := checkAndSort(items)
	if err != nil {
		t.Fatalf("checkAndSort failed: %v", err)
	}
	if err := importItems(db, pid, sorted, false); err != nil {
		t.Fatalf("importItems failed: %v", err)
	}
	if len(db.msgLists[pid]) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(db.msgLists[pid]))
	}
}
func TestImportFromJSONL_WithPidAndInlinePid(t *testing.T) {
	db := newFakeDB()
	// inline pid
	lines := []string{
		`{"pid":"p1","nonce":0,"msg":{"Id":"L1"},"assign":{"Id":"A1"}}`,
		`{"pid":"p1","nonce":1,"msg":{"Id":"L2"},"assign":{"Id":"A2"}}`,
	}
	// write temp jsonl
	var content string
	for _, l := range lines {
		content += l + "\n"
	}
	// create temp file
	path := t.TempDir() + "/data.jsonl"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write jsonl failed: %v", err)
	}
	// import using public function with db via wrapper
	// since ImportFromJSONL creates its own db, test importItems after parsing
	// simulate parse
	var parsed []syncSchema.ImportLine
	for _, l := range lines {
		var ln syncSchema.ImportLine
		if err := json.Unmarshal([]byte(l), &ln); err != nil {
			t.Fatalf("unmarshal line failed: %v", err)
		}
		parsed = append(parsed, ln)
	}
	items := make([]syncSchema.ImportItem, 0, len(parsed))
	for _, ln := range parsed {
		items = append(items, syncSchema.ImportItem{Nonce: ln.Nonce, Msg: ln.Msg, Assign: ln.Assign})
	}
	sorted, err := checkAndSort(items)
	if err != nil {
		t.Fatalf("checkAndSort failed: %v", err)
	}
	if err := importItems(db, "p1", sorted, false); err != nil {
		t.Fatalf("importItems failed: %v", err)
	}
	if len(db.msgLists["p1"]) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(db.msgLists["p1"]))
	}
}
