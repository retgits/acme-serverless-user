//+build mage

package main

import (
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/magefile/mage/sh"
	"gopkg.in/yaml.v2"
)

// ConfigRoot ...
type ConfigRoot struct {
	Config Config `yaml:"config"`
}

// Config ...
type Config struct {
	Stage   string `yaml:"stage"`
	Project string `yaml:"project"`
	Author  string `yaml:"author"`
	Team    string `yaml:"team"`
	App     string `yaml:"app"`
	Version string
	AWS     AWS
}

type AWS struct {
	Bucket string `yaml:"bucket"`
}

var config = getConfig()

func getConfig() Config {
	// Read the Pulumi YAML file for the stack
	fileContent, err := ioutil.ReadFile("mageconfig.yaml")
	if err != nil {
		panic(err)
	}
	source := string(fileContent)

	// Find all configuration variables that have been set as
	// environment variables
	re := regexp.MustCompile(`%{2}.*%{2}`)
	vars := re.FindAllString(source, -1)

	// Replace the configuration variables with the actual values
	// of the environment variable
	for _, v := range vars {
		source = strings.ReplaceAll(source, v, getEnvVar(strings.ReplaceAll(v, "%%", "")))
	}

	// Unmarshal the YAML content to a proper struct and return
	// the config part of it
	d := &ConfigRoot{}
	yaml.Unmarshal([]byte(source), d)

	// Update the values that haven't been set
	//d.Config.Version = gitVersion()

	return d.Config
}

func getEnvVar(envvar string) string {
	b, found := os.LookupEnv(envvar)
	if !found {
		return "unknown"
	}
	return b
}

func gitVersion() string {
	v, _ := sh.Output("git", "describe", "--tags", "--always", "--dirty=-dev")
	if len(v) == 0 {
		v = "dev"
	}
	return v
}

func appExists(app string) bool {
	for _, val := range apps {
		if val == app {
			return true
		}
	}
	return false
}