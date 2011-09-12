package main

import (
	"io/ioutil"
	"json"
	"os"
	)

type Config struct {
	//IRC stuff
	Nick string
	Pass string
	IRCServer string
	IRCDomain string
	IRCPort int
	SSL bool

	//Command access levels
	DefaultAccess []string
	AccessLevels map[string]AccessLevel
	Ignore []string

	//Backup related
	BackupLocation string
	ArchiverCommand string
	CopyCommand string
	BackupInterval int64

	//Map updater
	WorldTemp string
	MapUpdateCommand string
	MapUpdateInterval int64

	//MC Server config
	MCServerCommand string

	//Derived values:

	//The filename this config was pulled from
	source string
}

type AccessLevel struct {
	Members []string
	Allowed []string
}

func ReadConfig(confFile string) (*Config, os.Error) {
	raw, err := ioutil.ReadFile(confFile)
	if err != nil {
		return nil, err
	}

	conf := &Config{}

	if err = json.Unmarshal(raw, conf); err != nil {
		return nil, err
	}

	if err = sanityCheck(conf); err != nil {
		return nil, err
	}

	applyDefaults(conf)
	conf.source = confFile
	return conf, nil
}

func (c *Config) Reparse() os.Error {
	newConf, err := ReadConfig(c.source)
	if err != nil {
		return err
	}

	*c = *newConf

	return nil
}

func (c *Config) WriteConfig(confFile string) os.Error {
	tmp, err := ioutil.TempFile("", "mcbot-")
	if err != nil {
		return err
	}
	
	raw, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	err, _ = tmp.Write(raw)
	if err != nil {
		return err
	}

	tmp.Close()

	err = os.Rename(tmp.Name(), confFile)

	return err
}

func sanityCheck(c *Config) os.Error {
	return nil
}

func applyDefaults(c *Config) {
}