package domain

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Engine     string `yaml:"engine"`
	EnginePath string `yaml:"engine_path"`

	Char string `yaml:"char"`
	Name string `yaml:"name"`
}

func (c *Config) UnmarshalYAML(value *yaml.Node) error {
	type raw Config
	var tmp raw
	if err := value.Decode(&tmp); err != nil {
		return err
	}
	*c = Config(tmp)
	return nil
}

type TalentLevels struct {
	NA int
	E  int
	Q  int
}

func (t TalentLevels) String() string {
	return fmt.Sprintf("%d-%d-%d", t.NA, t.E, t.Q)
}
