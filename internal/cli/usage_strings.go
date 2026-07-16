package cli

const (
	usage_new_out    = "Output base directory for the generated project"
	usage_new_module = "Go module name (e.g. github.com/hymatrix/hype)"

	usage_get_package = "Go module path of the VMM package"

	usage_vmdocker_ref      = "VMDocker Git branch, tag, or reachable commit"
	usage_vmdocker_dir      = "Target VMDocker directory"
	usage_vmdocker_env_file = "Path to the .env file used for examples init"

	usage_vmdocker_profile_dir  = "Target agent profile directory"
	usage_vmdocker_profile_from = "Full base image name"

	usage_vmdocker_checkout_dir = "Target VMDocker V2 checkout"
	usage_vmdocker_profile      = "Path to profile.toml"
	usage_vmdocker_agent_bin    = "Path to vmdocker-agent binary"
	usage_vmdocker_node_url     = "Node URL"
	usage_vmdocker_private_key  = "Module signing private key"

	usage_vmdocker_module_id       = "VMDocker module id"
	usage_vmdocker_scheduler       = "Scheduler address"
	usage_vmdocker_runtime_type    = "Runtime type"
	usage_vmdocker_runtime_backend = "Runtime backend for spawn (docker or sandbox)"
	usage_vmdocker_env             = "Container environment assignment KEY=VALUE"
	usage_vmdocker_json            = "Print JSON output"
	usage_vmdocker_pid             = "Process id"

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
)
