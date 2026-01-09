package cli

const (
	usage_new_out    = "Output base directory for the generated project"
	usage_new_module = "Go module name (e.g. github.com/hymatrix/hype)"

	usage_get_package = "Go module path of the VMM package"

	usage_vmm_name   = "Name of the vmm"
	usage_vmm_format = "Module format of the vmm"

	usage_mount_name = "Name of the vmm"

	usage_module_name        = "Name of the module"
	usage_module_node_url    = "Node URL"
	usage_module_private_key = "Ethereum ECDSA secp256k1 private key hex (0x-prefixed)"

	usage_db_import_redis_url = "Redis URL"
	usage_db_import_file      = "JSON file path"
	usage_db_import_force     = "Override if data exists"

	usage_db_export_redis_url      = "Redis URL"
	usage_db_export_pid            = "Process id"
	usage_db_export_out            = "Output jsonl file path (supports .gz)"
	usage_db_export_progress_every = "Print progress every N lines"
)
