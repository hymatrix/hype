package cli

const (
	usage_new_out    = "Output base directory for the generated project"
	usage_new_module = "Go module name (e.g. github.com/hymatrix/hype)"

	usage_get_package = "Go module path of the VMM package"

	usage_vmdocker_version  = "VMDocker release tag"
	usage_vmdocker_dir      = "Target VMDocker directory"
	usage_vmdocker_env_file = "Path to the .env file used for examples init"

	usage_vmm_name   = "Name of the vmm"
	usage_vmm_format = "Module format of the vmm"

	usage_mount_name = "Name of the vmm"

	usage_module_name        = "Name of the module"
	usage_module_node_url    = "Node URL"
	usage_module_private_key = "Ethereum ECDSA secp256k1 private key hex (0x-prefixed)"

	usage_db_import_redis_url = "Redis URL"
	usage_db_import_file      = "JSONL file path"
	usage_db_import_force     = "Override if data exists"

	usage_db_export_redis_url      = "Redis URL"
	usage_db_export_pid            = "Process id (leave empty to export all)"
	usage_db_export_out            = "Output jsonl file path (supports .gz) or directory (if pid is empty)"
	usage_db_export_progress_every = "Print progress every N lines"
	usage_db_export_accid          = "Account id to list processes"

	usage_openclaw_node_url        = "Node URL"
	usage_openclaw_private_key     = "Ethereum ECDSA secp256k1 private key hex (0x-prefixed)"
	usage_openclaw_json            = "Print JSON output"
	usage_openclaw_module_id       = "Openclaw module id"
	usage_openclaw_scheduler       = "Scheduler address"
	usage_openclaw_model           = "Model name for spawn"
	usage_openclaw_provider        = "Provider name for spawn"
	usage_openclaw_api_key         = "Model API key"
	usage_openclaw_gateway_token   = "Openclaw gateway token"
	usage_openclaw_runtime_backend = "Runtime backend for spawn (docker or sandbox)"
	usage_openclaw_pid             = "Process id"
	usage_openclaw_bot_token       = "Telegram bot token"
	usage_openclaw_default_account = "Telegram default account"
	usage_openclaw_dm_policy       = "Telegram DM policy"
	usage_openclaw_allow_from      = "Telegram allow-from setting"
	usage_openclaw_code            = "Telegram pairing code"
	usage_openclaw_channel         = "Pairing channel"
	usage_openclaw_command         = "Chat command text"
	usage_openclaw_ui_listen       = "Listen address for Openclaw Web UI"
	usage_openclaw_ui_timeout_ms   = "Command timeout in milliseconds for Openclaw Web UI"
)
