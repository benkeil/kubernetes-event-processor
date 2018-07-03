// Copyright © 2018 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/benkeil/check-k8s/pkg/kube"
	hook "github.com/benkeil/kubernetes-event-processor/pkg/hooks"
	//"github.com/olivere/elastic"
	dotaccess "github.com/go-bongo/go-dotaccess"
	"github.com/sirupsen/logrus"
	"gopkg.in/olivere/elastic.v5"
	yaml "gopkg.in/yaml.v2"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
)

const (
	empty      = ""
	tab        = "  "
	logVersion = "v1"
)

var (
	log         = logrus.New()
	environment = os.Getenv("ENVIRONMENT")
	config      Config
)

type (
	// Config asd
	Config struct {
		Filter []Filter `json:"filter"`
	}

	// Filter asd
	Filter struct {
		Field string `json:"field"`
		Regex string `json:"regex"`
	}
)

func main() {
	run()
}

func init() {
	if os.Getenv("ENVIRONMENT") == "" {
		panic(fmt.Sprintf("cant find ENVIRONMENT environment variable"))
	}
	configureLogger()
	config = loadConfig()
}

func indexFunc() string {
	return fmt.Sprintf("%s.k8s.events.%s-%s", environment, logVersion, time.Now().Format("2006.01.02"))
}

func configureLogger() {
	client, err := elastic.NewClient(elastic.SetURL("http://elk01.i.sedorz.net:9200", "http://elk02.i.sedorz.net:9200"))
	if err != nil {
		panic(fmt.Sprintf("cant init elasticsearch client: %v", err))
	}
	hostname, _ := os.Hostname()
	hook, err := hook.NewElasticHookWithFunc(client, hostname, logrus.DebugLevel, indexFunc)
	if err != nil {
		panic(fmt.Sprintf("cant add hook: %v", err))
	}
	log.Hooks.Add(hook)
}

func run() {
	client, err := kube.GetKubeClient("production-admin")
	if err != nil {
		panic(fmt.Sprintf("cant get client: %v", err))
	}

	watchlist := cache.NewListWatchFromClient(
		client.CoreV1().RESTClient(),
		"events",
		v1.NamespaceAll,
		fields.Everything(),
	)

	_, controller := cache.NewInformer(
		watchlist,
		&v1.Event{},
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				filterEvent(obj.(*v1.Event))
			},
			DeleteFunc: func(obj interface{}) {
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
			},
		},
	)

	stop := make(chan struct{})
	defer close(stop)
	go controller.Run(stop)
	for {
		time.Sleep(time.Second)
	}
}

func filterEvent(event *v1.Event) {
	for _, filter := range config.Filter {
		value, err := dotaccess.Get(event, filter.Field)
		if err != nil {
			panic(fmt.Sprintf("cant access field: %v", err))
		}
		var regex = regexp.MustCompile(filter.Regex)
		if regex.MatchString(value.(string)) {
			processEvent(event)
			break
		}
	}
}

func processEvent(event *v1.Event) {
	in, _ := json.Marshal(event)
	data := make(map[string]interface{})
	json.Unmarshal(in, &data)
	log.WithFields(data).Info(fmt.Sprintf("%s [%s] %s.%s: %s \n", event.LastTimestamp, event.Reason, event.InvolvedObject.Name, event.InvolvedObject.Namespace, event.Message))
}

func prettyJSON(data interface{}) (string, error) {
	buffer := new(bytes.Buffer)
	encoder := json.NewEncoder(buffer)
	encoder.SetIndent(empty, tab)

	err := encoder.Encode(data)
	if err != nil {
		return empty, err
	}
	return buffer.String(), nil
}

func currentDir() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(fmt.Sprintf("can't get current dir: %v", err))
	}
	return dir
}

func loadYaml(path string) []byte {
	yamlString, err := ioutil.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("can't read file: %v", err))
	}
	return yamlString
}

func loadConfig() Config {
	config := Config{}

	dir := currentDir()
	valuesString := loadYaml(filepath.Join(dir, "config.yaml"))

	err := yaml.Unmarshal([]byte(valuesString), &config)
	if err != nil {
		panic(fmt.Sprintf("cant load config file: %v", err))
	}

	return config
}
