package driver

import (
	"time"
)

func GetSubConfig() (*SubConfig, error) {
	var config SubConfig
	exist, err := engine.Where("master = true").Get(&config)
	if !exist {
		_, _ = engine.Insert(&SubConfig{
			Master:      true,
			Version:     time.Now().Unix(),
			Ping:        0,
			LastVersion: time.Now().Unix(),
			BestNode:    "",
			ValidNode:   "",
		})
	}
	return &config, err
}

func UpdateSubConfig(u *SubConfig) error {
	_, err := engine.Update(u)
	if err != nil {
		return err
	}
	return nil
}

func AddPubConfig(config *PubConfig) error {
	_, err := engine.Insert(config)
	if err != nil {
		return err
	}
	return nil
}

func GetPubConfigList() ([]*PubConfig, error) {
	list := make([]*PubConfig, 0)
	err := engine.Find(&list)
	if err != nil {
		return nil, err
	}
	return list, nil
}
