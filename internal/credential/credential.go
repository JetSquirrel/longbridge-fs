package credential

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/longportapp/openapi-go/config"
)

// Load reads a key=value credential file and returns a config.Config.
func Load(path string) (*config.Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open credential file: %w", err)
	}
	defer f.Close()

	kv := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			kv[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read credential file: %w", err)
	}

	appKey := kv["api_key"]
	appSecret := kv["secret"]
	accessToken := kv["access_token"]

	if appKey == "" || appSecret == "" || accessToken == "" {
		return nil, fmt.Errorf("credential file missing required fields (api_key, secret, access_token)")
	}

	cfg, err := config.New(
		config.WithConfigKey(appKey, appSecret, accessToken),
	)
	if err != nil {
		return nil, fmt.Errorf("create config: %w", err)
	}

	return cfg, nil
}
