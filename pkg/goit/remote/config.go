package remote

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"souvik606/goit/pkg/goit/local"
	"strings"
)

type Config map[string]map[string]string

const goitDir = ".goit"

func findGoitDir(path string) (string, error) {
	parent := filepath.Dir(path)
	if path == parent {
		return "", fmt.Errorf("fatal: not a goit repository (or any of the parent directories): %s", goitDir)
	}

	goitPath := filepath.Join(path, goitDir)
	if _, err := os.Stat(goitPath); err == nil {
		fi, err := os.Stat(filepath.Join(goitPath, "config"))
		if err == nil && !fi.IsDir() {
			return goitPath, nil
		}
	}

	configPath := filepath.Join(path, "config")
	if _, err := os.Stat(configPath); err == nil {
		if local.IsValidBareRepo(path) {
			return path, nil
		}
	}

	return findGoitDir(parent)
}

func getConfigPath() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	basePath, err := findGoitDir(wd)
	if err != nil {
		return "", err
	}

	return filepath.Join(basePath, "config"), nil
}

func ReadConfig() (Config, error) {
	config := make(Config)
	configPath, err := getConfigPath()
	if err != nil {
		return nil, err
	}

	file, err := os.Open(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return config, nil
		}
		return nil, fmt.Errorf("opening config file %s: %w", configPath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var currentSection string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = line[1 : len(line)-1]
			if _, ok := config[currentSection]; !ok {
				config[currentSection] = make(map[string]string)
			}
			continue
		}

		if currentSection != "" && strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				config[currentSection][key] = value
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	return config, nil
}

func (c Config) Save() error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	var builder strings.Builder

	sections := make([]string, 0, len(c))
	for section := range c {
		sections = append(sections, section)
	}
	sort.Strings(sections)

	for _, section := range sections {
		fmt.Fprintf(&builder, "[%s]\n", section)

		keys := make([]string, 0, len(c[section]))
		for key := range c[section] {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for _, key := range keys {
			fmt.Fprintf(&builder, "\t%s = %s\n", key, c[section][key])
		}
	}

	err = os.WriteFile(configPath, []byte(builder.String()), 0644)
	if err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}
	return nil
}
