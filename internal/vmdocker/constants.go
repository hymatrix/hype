package vmdocker

const (
	RepoURL         = "https://github.com/cryptowizard0/vmdockerv2.git"
	repoURLNoSuffix = "https://github.com/cryptowizard0/vmdockerv2"
	repoURLSSH      = "git@github.com:cryptowizard0/vmdockerv2.git"
	DefaultRef      = "main"

	RedisContainer = "hype-vmdocker-redis"
	redisImage     = "redis:latest"
	redisAddress   = "127.0.0.1:6379"

	HealthURL    = "http://127.0.0.1:8080/info"
	NodeBinary   = "build/hymx-node"
	nodeCmdBin   = "hymx-node"
	nodeLockGlob = "hymx-*.lock"
	nodeLogGlob  = "hymx_*.log"
)
