package secrets

import (
	"testing"

	"github.com/mistweaverco/withsecrets/internal/config"
)

func TestInjectPathSecrets(t *testing.T) {
	t.Run("star injects without prefix", func(t *testing.T) {
		all := make(map[string]string)
		injectPathSecrets(all, "*", map[string]string{
			"USERNAME": "alice",
			"PASSWORD": "s3cret",
		})
		if all["USERNAME"] != "alice" || all["PASSWORD"] != "s3cret" {
			t.Fatalf("unexpected map: %#v", all)
		}
	})

	t.Run("named key prefixes", func(t *testing.T) {
		all := make(map[string]string)
		injectPathSecrets(all, "DB", map[string]string{
			"USERNAME": "alice",
			"PASSWORD": "s3cret",
		})
		if all["DB_USERNAME"] != "alice" || all["DB_PASSWORD"] != "s3cret" {
			t.Fatalf("unexpected map: %#v", all)
		}
	})
}

func TestResolveFetchKey(t *testing.T) {
	env := &config.Environment{
		Provider: "aws",
		Project:  "123",
		Region:   "eu-west-3",
	}

	t.Run("uses environment defaults", func(t *testing.T) {
		key := resolveFetchKey(env, config.EnvItem{})
		if key.provider != "aws" || key.project != "123" || key.region != "eu-west-3" {
			t.Fatalf("unexpected key: %#v", key)
		}
	})

	t.Run("item overrides region", func(t *testing.T) {
		key := resolveFetchKey(env, config.EnvItem{Region: "us-east-1"})
		if key.region != "us-east-1" {
			t.Fatalf("expected us-east-1, got %q", key.region)
		}
	})

	t.Run("aws empty project becomes default", func(t *testing.T) {
		key := resolveFetchKey(&config.Environment{Provider: "aws"}, config.EnvItem{})
		if key.project != "default" {
			t.Fatalf("expected default project, got %q", key.project)
		}
	})
}
