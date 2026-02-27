package syncer

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/hymatrix/hymx/db/rdb"
	nodeSchema "github.com/hymatrix/hymx/node/schema"
	schema "github.com/hymatrix/hype/internal/syncer/schema"
)

func ImportFromJSON(redisURL, jsonFile string, force bool) error {
	if redisURL == "" || jsonFile == "" {
		return errors.New("redis-url and file are required")
	}
	db := rdb.New(redisURL)
	defer db.Close()
	b, err := os.ReadFile(jsonFile)
	if err != nil {
		return err
	}
	var input schema.ImportPayload
	if err := json.Unmarshal(b, &input); err != nil {
		return err
	}
	sorted, err := checkAndSort(input.Items)
	if err != nil {
		return err
	}
	return importItems(db, input.Pid, sorted, force)
}

func ImportFromJSONL(redisURL, jsonlFile string, force bool) error {
	if redisURL == "" || jsonlFile == "" {
		return errors.New("redis-url and file are required")
	}
	db := rdb.New(redisURL)
	defer db.Close()
	f, err := os.Open(jsonlFile)
	if err != nil {
		return err
	}
	defer f.Close()
	var lines []schema.ImportLine
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		t := sc.Text()
		if t == "" {
			continue
		}
		var line schema.ImportLine
		if err := json.Unmarshal([]byte(t), &line); err != nil {
			return err
		}
		lines = append(lines, line)
	}
	if err := sc.Err(); err != nil {
		return err
	}
	if len(lines) == 0 {
		return nil
	}

	// group by pid
	groups := make(map[string][]schema.ImportItem)
	for _, ln := range lines {
		if ln.Pid == "" {
			return errors.New("pid is required in each jsonl line")
		}
		groups[ln.Pid] = append(groups[ln.Pid], schema.ImportItem{
			Nonce:  ln.Nonce,
			Msg:    ln.Msg,
			Assign: ln.Assign,
		})
	}

	for pid, items := range groups {
		sorted, err := checkAndSort(items)
		if err != nil {
			return fmt.Errorf("pid %s check failed: %w", pid, err)
		}
		if err := importItems(db, pid, sorted, force); err != nil {
			return fmt.Errorf("pid %s import failed: %w", pid, err)
		}
	}
	return nil
}

func checkAndSort(items []schema.ImportItem) ([]schema.ImportItem, error) {
	if len(items) == 0 {
		return items, nil
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Nonce < items[j].Nonce })
	if items[0].Nonce != 0 {
		return nil, fmt.Errorf("nonce must start at 0")
	}
	for i := 1; i < len(items); i++ {
		if items[i].Nonce != items[i-1].Nonce+1 {
			return nil, fmt.Errorf("nonce not continuous at position %d", i)
		}
	}
	return items, nil
}

func importItems(db nodeSchema.IDB, pid string, items []schema.ImportItem, force bool) error {
	for _, it := range items {
		exists := false
		if it.Msg.Id != "" {
			msg, err := db.GetMessage(it.Msg.Id)
			if err != nil {
				return err
			}
			if msg != nil {
				exists = true
			}
		}
		if exists && !force {
			continue
		}
		if err := db.Commit(pid, it.Nonce, it.Msg, it.Assign); err != nil {
			return err
		}
	}
	return nil
}
