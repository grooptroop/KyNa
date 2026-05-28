package config

import (
	"log"
	"os"
)

type Config struct {
	HttpAddr      string
	PostgresDSN   string
	K3sKubeconfig string
	HelmChartDir  string
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func Load() Config {
	cfg := Config{
		HttpAddr:      getenv("HTTP_ADDR", ":8080"),
		PostgresDSN:   getenv("POSTGRES_DSN", "postgres://kyna_user:postgres@localhost:5433/kyna?sslmode=disable"),
		K3sKubeconfig: getenv("K3S_KUBECONFIG", "/etc/rancher/k3s/k3s.yaml"),
		HelmChartDir:  getenv("HELM_CHART_DIR", "./helm/user-hello"),
	}

	if cfg.K3sKubeconfig != "" {
		if err := os.Setenv("KUBECONFIG", cfg.K3sKubeconfig); err != nil {
			log.Printf("failed to set KUBECONFIG: %v", err)
		}
	}

	return cfg
}
