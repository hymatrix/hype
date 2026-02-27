package syncer

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/hymatrix/hymx/db/rdb"
	nodeSchema "github.com/hymatrix/hymx/node/schema"
	schema "github.com/hymatrix/hype/internal/syncer/schema"
)

func ExportToJSONL(redisURL, pid, outFile string, opts *schema.ExportOptions) error {
	if redisURL == "" || pid == "" || outFile == "" {
		return errors.New("redis-url, pid and outFile are required")
	}
	db := rdb.New(redisURL)
	defer db.Close()
	return exportToJSONLWithDB(db, pid, outFile, opts)
}

func ExportAllToJSONL(redisURL, outFile string, opts *schema.ExportOptions) error {
	if redisURL == "" || outFile == "" {
		return errors.New("redis-url and out are required")
	}

	var progressEvery int64
	var progress func(done, total int64)
	if opts != nil {
		progressEvery = opts.ProgressEvery
		progress = opts.Progress
	}

	db := rdb.New(redisURL)
	defer db.Close()

	pids, _, err := db.GetAllProcess()
	if err != nil {
		return err
	}
	if len(pids) == 0 {
		return nil
	}

	f, err := os.Create(outFile)
	if err != nil {
		return err
	}
	defer f.Close()

	if strings.HasSuffix(outFile, ".gz") {
		gw := gzip.NewWriter(f)
		defer gw.Close()
		for _, p := range pids {
			if err := exportLinesWithProgress(db, p, gw, progressEvery, progress); err != nil {
				return err
			}
		}
		return nil
	}
	for _, p := range pids {
		if err := exportLinesWithProgress(db, p, f, progressEvery, progress); err != nil {
			return err
		}
	}
	return nil
}

func exportToJSONLWithDB(db nodeSchema.IDB, pid, outFile string, opts *schema.ExportOptions) error {
	var progressEvery int64
	var progress func(done, total int64)
	if opts != nil {
		progressEvery = opts.ProgressEvery
		progress = opts.Progress
	}

	f, err := os.Create(outFile)
	if err != nil {
		return err
	}
	defer f.Close()

	if strings.HasSuffix(outFile, ".gz") {
		gw := gzip.NewWriter(f)
		defer gw.Close()
		return exportLinesWithProgress(db, pid, gw, progressEvery, progress)
	}
	return exportLinesWithProgress(db, pid, f, progressEvery, progress)
}

func exportLinesWithProgress(db nodeSchema.IDB, pid string, w io.Writer, progressEvery int64, progress func(done, total int64)) error {
	if pid == "" {
		return errors.New("pid is required")
	}
	lastNonce, err := db.GetNonce(pid)
	if err != nil {
		return err
	}
	if lastNonce < 0 {
		return nil
	}
	total := lastNonce + 1
	if progressEvery <= 0 {
		progressEvery = 1000
	}
	bw := bufio.NewWriter(w)
	for nonce := int64(0); nonce <= lastNonce; nonce++ {
		msg, err := db.GetMessageByNonce(pid, nonce)
		if err != nil {
			return errors.New("get message by nonce failed: " + ", nonce: " + strconv.FormatInt(nonce, 10) + err.Error())
		}
		if msg == nil {
			return errors.New("msg is nil, nonce: " + strconv.FormatInt(nonce, 10))
		}
		assign, err := db.GetAssignByNonce(pid, nonce)
		if err != nil {
			return err
		}
		if assign == nil {
			return errors.New("assign is nil, nonce: " + strconv.FormatInt(nonce, 10))
		}
		line := schema.ImportLine{
			Pid:    pid,
			Nonce:  nonce,
			Msg:    *msg,
			Assign: *assign,
		}
		b, err := json.Marshal(line)
		if err != nil {
			return err
		}
		if _, err := bw.Write(append(b, '\n')); err != nil {
			return err
		}
		if progress != nil {
			done := nonce + 1
			if done == total || done%progressEvery == 0 {
				progress(done, total)
			}
		}
	}
	return bw.Flush()
}
