package config

import (
	"os"
)

type Config struct {
	HttpAddr      string
	PostgresDSN   string
	K3sKubeconfig string // уже не используем, но можно оставить на будущее
	HelmChartDir  string
}

func Load() Config {
	cfg := Config{
		HttpAddr:      getenv("HTTP_ADDR", ":8080"),
		PostgresDSN:   getenv("POSTGRES_DSN", "postgres://kyna_user:postgres@localhost:5432/kyna?sslmode=disable"),
		K3sKubeconfig: getenv("K3S_KUBECONFIG", "/etc/rancher/k3s/k3s.yaml"),
		HelmChartDir:  getenv("HELM_CHART_DIR", "./helm/user-hello"),
	}
	return cfg
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
