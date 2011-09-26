package main

import (
	"io/ioutil"
	"json"
	"os"
	)

type Config struct {
	HostOS string

	//IRC stuff
	Nick string
	Pass string
	AttnChar string
	IrcServer string
	IrcChan string
	IrcChanKey string
	IrcDomain string
	IrcPort int
	SSL bool

	//Command access levels
	DefaultAccess []string
	AccessLevels map[string]AccessLevel
	Ignore []string

	//Backup related
	BackupCommand cmd
	BackupInterval int64

	//Map updater
	MapUpdateCommand cmd
	MapUpdateInterval int64

	//MC Server config
	MCServerCommand cmd
	MCServerDir string

	//Derived values:
	defaultAccess map[string]bool
	accessLevels map[string]map[string]bool
	accessLevelMembers map[string][]string
	ignore map[string]bool

	//The filename this config was pulled from
	source string
}

type AccessLevel struct {
	Members []string
	Allowed []string
}

type cmd struct {
	Command string
	Args []string
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
	mungeConfig(conf)
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

	_, err = tmp.Write(raw)
	if err != nil {
		return err
	}

	tmp.Close()

	err = os.Rename(tmp.Name(), confFile)

	return err
}

//Munge the config file/json friendly constructs into easier to use formats
func mungeConfig(conf *Config) {
	conf.defaultAccess = make(map[string]bool, len(conf.DefaultAccess))
	for _, cmd := range conf.DefaultAccess {
		conf.defaultAccess[cmd] = true
	}

	conf.ignore = make(map[string]bool, len(conf.Ignore))
	for _, nick := range conf.Ignore {
		conf.ignore[nick] = true
	}

	conf.accessLevels = make(map[string]map[string]bool, len(conf.AccessLevels))
	conf.accessLevelMembers = make(map[string][]string)
	for title, level := range conf.AccessLevels {
		for _, nick := range level.Members {
			conf.accessLevelMembers[nick] = append(conf.accessLevelMembers[nick], title)
		}

		conf.accessLevels[title] = make(map[string]bool)
		for _, cmd := range level.Allowed {
			conf.accessLevels[title][cmd] = true
		}
	}
}


func sanityCheck(c *Config) os.Error {
	return nil
}

func applyDefaults(c *Config) {
	return
}