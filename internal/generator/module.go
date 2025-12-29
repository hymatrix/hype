package generator

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/everFinance/goether"
	hymxSchema "github.com/hymatrix/hymx/schema"
	"github.com/hymatrix/hymx/sdk"
	"github.com/hymatrix/hype/internal/generator/schema"
	"github.com/permadao/goar"
)

func GenAndSaveModule(options schema.Options) error {
	signer, err := goether.NewSigner(options.PrivateKey)
	if err != nil {
		return err
	}
	bundler, err := goar.NewBundler(signer)
	if err != nil {
		return err
	}

	module := hymxSchema.Module{
		Base:         hymxSchema.DefaultBaseModule,
		ModuleFormat: options.ModuleFormat,
	}

	hymxSdk := sdk.NewFromBundler(options.NodeUrl, bundler)
	itemId, err := hymxSdk.SaveModule([]byte{}, module)
	if err != nil {
		return err
	}

	fmt.Println("module itemId:", itemId)
	src := fmt.Sprintf("mod-%s.json", itemId)
	dstDir := filepath.Join(options.ProjectDir, "cmd", "mod")
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return err
	}
	dst := filepath.Join(dstDir, src)
	if err := os.Rename(src, dst); err != nil {
		return err
	}
	return nil
}
