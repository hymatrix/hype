package schema

import goarSchema "github.com/permadao/goar/schema"

type ImportItem struct {
	Nonce  int64                 `json:"nonce"`
	Msg    goarSchema.BundleItem `json:"msg"`
	Assign goarSchema.BundleItem `json:"assign"`
}

type ImportPayload struct {
	Pid   string       `json:"pid"`
	Items []ImportItem `json:"items"`
}

type ImportLine struct {
	Pid    string                `json:"pid"`
	Nonce  int64                 `json:"nonce"`
	Msg    goarSchema.BundleItem `json:"msg"`
	Assign goarSchema.BundleItem `json:"assign"`
}

type ExportOptions struct {
	ProgressEvery int64
	Progress      func(done, total int64)
}
