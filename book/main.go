package main

import (
	"flag"
	"log"
	"os"

	"gopkg.in/yaml.v3"

	"kubelearn/book/app"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "app.yml", "Path to config file")
	flag.Parse()

	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalln(err)
	}

	var cfg app.Config
	if err = yaml.Unmarshal(data, &cfg); err != nil {
		log.Fatalln(err)
	}

	if err = app.SetUp(&cfg); err != nil {
		log.Fatalln(err)
	}
}
