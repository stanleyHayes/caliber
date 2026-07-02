package alertmanager

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestAlertmanagerConfigIsValidYAML(t *testing.T) {
	b, err := os.ReadFile(filepath.Clean("alertmanager.yml"))
	require.NoError(t, err)

	var cfg struct {
		Route struct {
			Receiver string `yaml:"receiver"`
		} `yaml:"route"`
		Receivers []struct {
			Name string `yaml:"name"`
		} `yaml:"receivers"`
	}
	require.NoError(t, yaml.Unmarshal(b, &cfg))
	assert.Equal(t, "local-webhook", cfg.Route.Receiver)
	require.Len(t, cfg.Receivers, 1)
	assert.Equal(t, "local-webhook", cfg.Receivers[0].Name)
}

func TestPrometheusAlertRulesAreValidYAML(t *testing.T) {
	b, err := os.ReadFile(filepath.Clean("../prometheus/alerts.yml"))
	require.NoError(t, err)

	var ruleFile struct {
		Groups []struct {
			Name  string `yaml:"name"`
			Rules []struct {
				Alert       string `yaml:"alert"`
				Expr        string `yaml:"expr"`
				For         string `yaml:"for"`
				Labels      map[string]string `yaml:"labels"`
				Annotations map[string]string `yaml:"annotations"`
			} `yaml:"rules"`
		} `yaml:"groups"`
	}
	require.NoError(t, yaml.Unmarshal(b, &ruleFile))

	wantAlerts := map[string]bool{
		"CaliberAPIDown":              false,
		"CaliberWorkerDown":           false,
		"CaliberHighHTTPLatency":      false,
		"CaliberHighHTTPErrorRate":    false,
		"CaliberHighAIFailureRate":    false,
		"CaliberHighQueueJobErrorRate": false,
	}
	for _, g := range ruleFile.Groups {
		assert.NotEmpty(t, g.Name)
		for _, r := range g.Rules {
			assert.NotEmpty(t, r.Alert)
			assert.NotEmpty(t, r.Expr)
			assert.NotEmpty(t, r.For)
			assert.NotEmpty(t, r.Labels["severity"])
			assert.NotEmpty(t, r.Annotations["summary"])
			if _, ok := wantAlerts[r.Alert]; ok {
				wantAlerts[r.Alert] = true
			}
		}
	}
	for name, found := range wantAlerts {
		assert.True(t, found, "expected alert %q not found", name)
	}
	assert.Contains(t, string(b), "up{job=\"caliber-api\"}")
}
