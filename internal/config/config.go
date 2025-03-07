package config

import (
	"bytes"
	"encoding/json"
	"os"
	"path"
)

type Config struct {
	Db_url            string `json:"db_url"`
	Current_user_name string `json:"current_user_name"`
}

const config_file_name = ".gatorconfig.json"

func Read() (*Config, error) {
	path, err := get_config_file_path()
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	c := &Config{}
	dec := json.NewDecoder(f)
	if err := dec.Decode(c); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Config) SetUser(name string) error {
	c.Current_user_name = name
	if err := c.write(); err != nil {
		return err
	}
	return nil
}

func (c *Config) write() error {
	path, err := get_config_file_path()
	if err != nil {
		return err
	}

	bs, err := json.Marshal(c)
	if err != nil {
		return err
	}
	var j bytes.Buffer
	err = json.Indent(&j, bs, "", "  ")

	err = os.WriteFile(path, j.Bytes(), 0666)
	if err != nil {
		return err
	}

	return nil
}

func get_config_file_path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return path.Join(home, config_file_name), nil
}
