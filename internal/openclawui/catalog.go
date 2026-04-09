package openclawui

import (
	"fmt"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	commandCatalogFactoryMu sync.RWMutex
	commandCatalogFactory   catalogFactory
)

type commandOverlay struct {
	Supported           bool
	DisabledReason      string
	ImportedEnvRequired bool
}

type fieldOverlay struct {
	Skip     bool
	Kind     string
	Required bool
	Options  []fieldOption
	EnvKeys  []string
	Group    string
}

func SetCommandCatalogFactory(factory func() *cobra.Command) {
	commandCatalogFactoryMu.Lock()
	defer commandCatalogFactoryMu.Unlock()
	commandCatalogFactory = factory
}

func buildCatalog() ([]catalogRoot, error) {
	root := getCatalogRootCommand()
	if root == nil {
		return nil, fmt.Errorf("command catalog factory is not configured")
	}

	commands := root.Commands()
	roots := make([]catalogRoot, 0, len(commands))
	for _, cmd := range commands {
		if shouldSkipCommand(cmd) {
			continue
		}

		rootEntry := catalogRoot{
			Name:        cmd.Name(),
			Title:       humanizeName(cmd.Name()),
			Description: strings.TrimSpace(cmd.Short),
		}

		subcommands := visibleSubcommands(cmd)
		if len(subcommands) == 0 {
			leaf, err := buildCatalogCommand([]string{cmd.Name()}, cmd)
			if err != nil {
				return nil, err
			}
			rootEntry.Commands = []catalogCommand{leaf}
		} else {
			rootEntry.Commands = make([]catalogCommand, 0, len(subcommands))
			for _, subcmd := range subcommands {
				leaf, err := buildCatalogCommand([]string{cmd.Name(), subcmd.Name()}, subcmd)
				if err != nil {
					return nil, err
				}
				rootEntry.Commands = append(rootEntry.Commands, leaf)
			}
		}
		if rootEntry.Commands == nil {
			rootEntry.Commands = []catalogCommand{}
		}

		roots = append(roots, rootEntry)
	}

	return roots, nil
}

func buildCatalogCommand(path []string, cmd *cobra.Command) (catalogCommand, error) {
	fields, err := collectCatalogFields(path, cmd)
	if err != nil {
		return catalogCommand{}, err
	}
	if fields == nil {
		fields = []catalogField{}
	}

	overlay := commandOverlayFor(path)
	return catalogCommand{
		Name:                path[len(path)-1],
		Title:               humanizeName(path[len(path)-1]),
		Path:                path,
		Description:         strings.TrimSpace(cmd.Short),
		Supported:           overlay.Supported,
		DisabledReason:      overlay.DisabledReason,
		ImportedEnvRequired: overlay.ImportedEnvRequired,
		Fields:              fields,
	}, nil
}

func collectCatalogFields(path []string, cmd *cobra.Command) ([]catalogField, error) {
	var fields []catalogField
	seen := map[string]struct{}{}

	appendSet := func(flagSet *pflag.FlagSet, group string) error {
		if flagSet == nil {
			return nil
		}
		flagSet.VisitAll(func(flag *pflag.Flag) {
			if flag == nil {
				return
			}
			if _, ok := seen[flag.Name]; ok {
				return
			}
			if shouldSkipFlag(flag.Name) {
				return
			}

			overlay := fieldOverlayFor(path, flag.Name)
			if overlay.Skip {
				return
			}

			field, ok := buildCatalogField(path, flag, overlay, group)
			if !ok {
				return
			}
			seen[flag.Name] = struct{}{}
			fields = append(fields, field)
		})
		return nil
	}

	if err := appendSet(cmd.InheritedFlags(), "shared"); err != nil {
		return nil, err
	}
	if err := appendSet(cmd.NonInheritedFlags(), "command"); err != nil {
		return nil, err
	}

	return fields, nil
}

func buildCatalogField(path []string, flag *pflag.Flag, overlay fieldOverlay, group string) (catalogField, bool) {
	kind := overlay.Kind
	if kind == "" {
		kind = normalizeFlagKind(flag.Value.Type())
	}
	if kind == "" {
		return catalogField{}, false
	}

	required := overlay.Required
	if !required {
		_, required = flag.Annotations[cobra.BashCompOneRequiredFlag]
	}

	groupName := overlay.Group
	if groupName == "" {
		groupName = group
	}

	return catalogField{
		Name:         flag.Name,
		Label:        humanizeName(flag.Name),
		Description:  strings.TrimSpace(flag.Usage),
		Kind:         kind,
		DefaultValue: flag.DefValue,
		Required:     required,
		Options:      overlay.Options,
		EnvKeys:      overlay.EnvKeys,
		Group:        groupName,
	}, true
}

func getCatalogRootCommand() *cobra.Command {
	commandCatalogFactoryMu.RLock()
	factory := commandCatalogFactory
	commandCatalogFactoryMu.RUnlock()
	if factory == nil {
		return nil
	}
	return factory()
}

func findCatalogCommand(roots []catalogRoot, path []string) *catalogCommand {
	if len(path) == 0 {
		return nil
	}
	for i := range roots {
		if roots[i].Name != path[0] {
			continue
		}
		for j := range roots[i].Commands {
			if equalPath(roots[i].Commands[j].Path, path) {
				return &roots[i].Commands[j]
			}
		}
	}
	return nil
}

func visibleSubcommands(cmd *cobra.Command) []*cobra.Command {
	children := cmd.Commands()
	if len(children) == 0 {
		return nil
	}

	visible := make([]*cobra.Command, 0, len(children))
	for _, child := range children {
		if shouldSkipCommand(child) {
			continue
		}
		visible = append(visible, child)
	}
	return visible
}

func equalPath(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func shouldSkipCommand(cmd *cobra.Command) bool {
	if cmd == nil {
		return true
	}
	if cmd.Hidden {
		return true
	}
	switch cmd.Name() {
	case "help", "completion":
		return true
	default:
		return false
	}
}

func shouldSkipFlag(name string) bool {
	switch name {
	case "help", "json", "env-file":
		return true
	default:
		return false
	}
}

func commandOverlayFor(path []string) commandOverlay {
	joined := strings.Join(path, "/")
	switch joined {
	case "ui":
		return commandOverlay{
			Supported:      false,
			DisabledReason: "The Web UI cannot launch another embedded Web UI instance from inside itself.",
		}
	case "repl":
		return commandOverlay{
			Supported:      false,
			DisabledReason: "The REPL is terminal-interactive and is not executable from the embedded Web UI.",
		}
	case "vmdocker/init":
		return commandOverlay{
			Supported:           true,
			ImportedEnvRequired: true,
		}
	default:
		return commandOverlay{Supported: true}
	}
}

func fieldOverlayFor(path []string, flagName string) fieldOverlay {
	joined := strings.Join(path, "/")
	overlay := genericFieldOverlay(flagName)

	merge := func(next fieldOverlay) fieldOverlay {
		overlay = mergeFieldOverlay(overlay, next)
		return overlay
	}

	switch joined {
	case "new":
		if flagName == "module" {
			return merge(fieldOverlay{Required: true})
		}
	case "get":
		if flagName == "package" {
			return merge(fieldOverlay{Required: true})
		}
	case "vmm":
		if flagName == "name" || flagName == "format" {
			return merge(fieldOverlay{Required: true})
		}
	case "mount":
		if flagName == "name" {
			return merge(fieldOverlay{Required: true})
		}
	case "module":
		if flagName == "name" {
			return merge(fieldOverlay{Required: true})
		}
	case "db-import":
		if flagName == "redis-url" || flagName == "file" {
			return merge(fieldOverlay{Required: true, Kind: mapPathKind(flagName)})
		}
	case "db-export":
		if flagName == "redis-url" || flagName == "out" {
			return merge(fieldOverlay{Required: true, Kind: mapPathKind(flagName)})
		}
	case "run":
		if flagName == "mode" {
			return merge(fieldOverlay{
				Kind: "enum",
				Options: []fieldOption{
					{Label: "normal", Value: "normal"},
					{Label: "rebuild", Value: "rebuild"},
				},
			})
		}
	case "openclaw/spawn":
		switch flagName {
		case "module-id":
			return merge(fieldOverlay{Required: true, EnvKeys: []string{"OPENCLAW_MODULE_ID"}})
		case "scheduler":
			return merge(fieldOverlay{Required: true, EnvKeys: []string{"VMDOCKER_SCHEDULER"}})
		case "model":
			return merge(fieldOverlay{EnvKeys: []string{"OPENCLAW_MODEL"}})
		case "provider":
			return merge(fieldOverlay{EnvKeys: []string{"OPENCLAW_PROVIDER"}})
		case "api-key":
			return merge(fieldOverlay{Kind: "secret", EnvKeys: []string{"OPENCLAW_API_KEY"}})
		case "gateway-token":
			return merge(fieldOverlay{Required: true, Kind: "secret", EnvKeys: []string{"OPENCLAW_GATEWAY_TOKEN"}})
		case "runtime-backend":
			return merge(fieldOverlay{
				Kind: "enum",
				Options: []fieldOption{
					{Label: "auto", Value: ""},
					{Label: "docker", Value: "docker"},
					{Label: "sandbox", Value: "sandbox"},
				},
			})
		case "bot-token":
			return merge(fieldOverlay{Kind: "secret", EnvKeys: []string{"OPENCLAW_TELEGRAM_BOT_TOKEN"}})
		case "default-account":
			return merge(fieldOverlay{EnvKeys: []string{"OPENCLAW_TELEGRAM_DEFAULT_ACCOUNT"}})
		case "dm-policy":
			return merge(dmPolicyOverlay(true))
		case "allow-from":
			return merge(fieldOverlay{EnvKeys: []string{"OPENCLAW_TELEGRAM_ALLOW_FROM"}})
		}
	case "openclaw/conf-tg":
		switch flagName {
		case "pid", "bot-token":
			if flagName == "bot-token" {
				return merge(fieldOverlay{Required: true, Kind: "secret", EnvKeys: []string{"OPENCLAW_TELEGRAM_BOT_TOKEN"}})
			}
			return merge(fieldOverlay{Required: true})
		case "default-account":
			return merge(fieldOverlay{EnvKeys: []string{"OPENCLAW_TELEGRAM_DEFAULT_ACCOUNT"}})
		case "dm-policy":
			return merge(dmPolicyOverlay(true))
		case "allow-from":
			return merge(fieldOverlay{EnvKeys: []string{"OPENCLAW_TELEGRAM_ALLOW_FROM"}})
		}
	case "openclaw/pair-tg":
		switch flagName {
		case "pid", "code":
			return merge(fieldOverlay{Required: true})
		case "channel":
			return merge(fieldOverlay{
				Kind: "enum",
				Options: []fieldOption{
					{Label: "telegram", Value: "telegram"},
				},
			})
		case "dm-policy":
			return merge(dmPolicyOverlay(true))
		}
	case "openclaw/chat":
		switch flagName {
		case "pid", "command":
			return merge(fieldOverlay{Required: true, Kind: mapCommandKind(flagName)})
		}
	case "claude/spawn":
		switch flagName {
		case "module-id":
			return merge(fieldOverlay{Required: true, EnvKeys: []string{"VMDOCKER_MODULE_ID"}})
		case "scheduler":
			return merge(fieldOverlay{Required: true, EnvKeys: []string{"VMDOCKER_SCHEDULER"}})
		case "api-key":
			return merge(fieldOverlay{Required: true, Kind: "secret", EnvKeys: []string{"ANTHROPIC_API_KEY"}})
		case "base-url":
			return merge(fieldOverlay{EnvKeys: []string{"ANTHROPIC_BASE_URL"}})
		case "model":
			return merge(fieldOverlay{EnvKeys: []string{"ANTHROPIC_MODEL"}})
		case "code-flags":
			return merge(fieldOverlay{Kind: "multiline", EnvKeys: []string{"CLAUDE_CODE_FLAGS"}})
		case "runtime-backend":
			return merge(fieldOverlay{
				Kind: "enum",
				Options: []fieldOption{
					{Label: "auto", Value: ""},
					{Label: "docker", Value: "docker"},
					{Label: "sandbox", Value: "sandbox"},
				},
				EnvKeys: []string{"RUNTIME_BACKEND"},
			})
		}
	case "claude/chat":
		switch flagName {
		case "pid", "command":
			return merge(fieldOverlay{Required: true, Kind: mapCommandKind(flagName)})
		}
	case "claude/exec":
		switch flagName {
		case "pid":
			return merge(fieldOverlay{Required: true})
		case "prompt":
			return merge(fieldOverlay{Required: true, Kind: "multiline"})
		}
	case "vmdocker/get":
		if flagName == "dir" {
			return merge(fieldOverlay{Kind: "path"})
		}
	}

	return overlay
}

func genericFieldOverlay(flagName string) fieldOverlay {
	switch flagName {
	case "private-key":
		return fieldOverlay{
			Kind:     "secret",
			EnvKeys:  []string{"HYPE_PRIVATE_KEY", "PRV_KEY", "VMDOCKER_PRIVATE_KEY"},
			Group:    "shared",
			Required: false,
		}
	case "api-key", "gateway-token", "bot-token":
		return fieldOverlay{Kind: "secret"}
	case "dir", "file", "out":
		return fieldOverlay{Kind: "path"}
	case "command":
		return fieldOverlay{Kind: "multiline"}
	case "prompt":
		return fieldOverlay{Kind: "multiline"}
	case "redis-url":
		return fieldOverlay{
			EnvKeys: []string{"REDIS_URL"},
		}
	default:
		return fieldOverlay{}
	}
}

func mergeFieldOverlay(base, next fieldOverlay) fieldOverlay {
	merged := base
	merged.Skip = merged.Skip || next.Skip
	if next.Kind != "" {
		merged.Kind = next.Kind
	}
	merged.Required = merged.Required || next.Required
	if len(next.Options) > 0 {
		merged.Options = next.Options
	}
	if len(next.EnvKeys) > 0 {
		merged.EnvKeys = next.EnvKeys
	}
	if next.Group != "" {
		merged.Group = next.Group
	}
	return merged
}

func dmPolicyOverlay(includeEnv bool) fieldOverlay {
	overlay := fieldOverlay{
		Kind: "enum",
		Options: []fieldOption{
			{Label: "open", Value: "open"},
			{Label: "pairing", Value: "pairing"},
		},
	}
	if includeEnv {
		overlay.EnvKeys = []string{"OPENCLAW_TELEGRAM_DM_POLICY"}
	}
	return overlay
}

func mapPathKind(flagName string) string {
	switch flagName {
	case "file", "out":
		return "path"
	default:
		return ""
	}
}

func mapCommandKind(flagName string) string {
	if flagName == "command" {
		return "multiline"
	}
	return ""
}

func normalizeFlagKind(flagType string) string {
	switch flagType {
	case "string":
		return "string"
	case "bool":
		return "bool"
	case "int":
		return "int"
	case "int64":
		return "int64"
	default:
		return ""
	}
}

func humanizeName(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}

	parts := strings.Split(value, "-")
	for i := range parts {
		if parts[i] == "" {
			continue
		}
		switch strings.ToLower(parts[i]) {
		case "id":
			parts[i] = "ID"
		case "pid":
			parts[i] = "PID"
		case "url":
			parts[i] = "URL"
		case "api":
			parts[i] = "API"
		case "tg":
			parts[i] = "TG"
		case "ui":
			parts[i] = "UI"
		default:
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, " ")
}
