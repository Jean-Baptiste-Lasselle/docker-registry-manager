package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/astaxie/beego"
	"github.com/sirupsen/logrus"
	"github.com/snagles/docker-registry-manager/app/conf"
	"github.com/snagles/docker-registry-manager/app/models"
	_ "github.com/snagles/docker-registry-manager/app/routers"
	"github.com/spf13/viper"
)

const (
	appVersion = "2.0.1"
)

func main() {
	config := flag.String("config", "", "config path location")
	version := flag.Bool("version", false, "app version")
	flag.Parse()

	if *version == true {
		fmt.Println("Version: " + appVersion)
		os.Exit(1)
	}
	c, err := parseConfig(*config)
	if err != nil {
		logrus.Fatal(err)
	}

	err = setlevel(c.App.LogLevel)
	if err != nil {
		logrus.Fatal(err)
	}

	for _, r := range c.Registries {
		if r.URL != "" {
			url, err := url.Parse(r.URL)
			if err != nil {
				logrus.Fatalf("Failed to parse registry from the passed url (%s): %s", r.URL, err)
			}
			port, err := strconv.Atoi(url.Port())
			if err != nil || port == 0 {
				logrus.Fatalf("Failed to add registry (%s), invalid port: %s", r.URL, err)
			}
			duration, err := time.ParseDuration(r.RefreshRate)
			if err != nil {
				logrus.Fatalf("Failed to add registry (%s), invalid duration: %s", r.URL, err)
			}
			if r.Password != "" && r.Username != "" {
				if _, err := manager.AddRegistry(url.Scheme, url.Hostname(), r.Username, r.Password, port, duration, r.SkipTLS); err != nil {
					logrus.Fatalf("Failed to add registry (%s): %s", r.URL, err)
				}
			} else {
				if _, err := manager.AddRegistry(url.Scheme, url.Hostname(), "", "", port, duration, r.SkipTLS); err != nil {
					logrus.Fatalf("Failed to add registry (%s): %s", r.URL, err)
				}
			}
		}
	}

	// Beego configuration
	beego.BConfig.AppName = "docker-registry-manager"
	beego.BConfig.RunMode = "dev"
	beego.BConfig.Listen.EnableAdmin = true
	beego.BConfig.CopyRequestBody = true

	// add template functions
	beego.AddFuncMap("shortenDigest", DigestShortener)
	beego.AddFuncMap("statToSeconds", StatToSeconds)
	beego.AddFuncMap("bytefmt", ByteFmt)
	beego.AddFuncMap("bytefmtdiff", ByteDiffFmt)
	beego.AddFuncMap("timeAgo", TimeAgo)
	beego.AddFuncMap("oneIndex", func(i int) int { return i + 1 })
	beego.Run()
}

func setlevel(level string) error {
	switch {
	case level == "panic":
		logrus.SetLevel(logrus.PanicLevel)
	case level == "fatal":
		logrus.SetLevel(logrus.FatalLevel)
	case level == "error":
		logrus.SetLevel(logrus.ErrorLevel)
	case level == "warn":
		logrus.SetLevel(logrus.WarnLevel)
	case level == "info":
		logrus.SetLevel(logrus.InfoLevel)
	case level == "debug":
		logrus.SetLevel(logrus.DebugLevel)
	default:
		return fmt.Errorf("Unrecognized log level: %s", level)
	}
	return nil
}

type config struct {
	App struct {
		LogLevel string `mapstructure:"log-level"`
		Port     int
	}
	Registries map[string]struct {
		URL         string
		Username    string
		Password    string
		SkipTLS     bool   `mapstructure:"skip-tls-validation"`
		RefreshRate string `mapstructure:"refresh-rate"`
	} `mapstructure:"registries"`
}

func parseConfig(configPath string) (*config, error) {
	v := viper.New()

	// If the config path is not passed use the default project dir
	if configPath != "" {
		v.AddConfigPath(path.Dir(configPath))
		base := path.Base(configPath)
		ext := path.Ext(configPath)
		v.SetConfigName(base[0 : len(base)-len(ext)])
	} else {
		// use the default tree
		v.SetConfigName("config")
		v.AddConfigPath(conf.GOPATH + "/src/github.com/snagles/docker-registry-manager")
		v.AddConfigPath("../")
		v.AddConfigPath("/opt/docker-registry-manager")
	}

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("Failed to read in config file: %s", err)
	}

	c := config{}
	if err := v.Unmarshal(&c); err != nil {
		return nil, fmt.Errorf("Unable to unmarshal config file: %s", err)
	}
	return &c, nil
}
