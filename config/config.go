package config

import (
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"sync"
)

// Environments 项目主要配置项[子项] 如果需要扩展 在这里添加结构来实现yaml的解析
type Environments struct {
	ProjectName string         `yaml:"project_name"` //项目名称
	Port        string         `yaml:"port"`         //服务运行的 :port
	HTTPS       `yaml:"https"` //https配置
}

type HTTPS struct {
	Enable  bool   `yaml:"enable"`
	CrtFile string `yaml:"crt_path"`
	KeyFile string `yaml:"key_path"`
}

var (
	std      *Environments
	loadOnce sync.Once
)

func GetConfig() *Environments {
	loadOnce.Do(func() {
		std = new(Environments)
		yamlFile, err := ioutil.ReadFile("./config.yaml")
		if err != nil {
			panic(err)
		}
		err = yaml.Unmarshal(yamlFile, std)
		if err != nil {
			//读取配置文件失败,停止执行
			panic("read config file error:" + err.Error())
		}
	})
	return std
}
