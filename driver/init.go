package driver

import (
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

// SubConfig 订阅信息
type SubConfig struct {
	Master      bool   `json:"master"` // 是否需要用该配置
	Version     int64  `json:"version,omitempty"`
	Ping        int64  `json:"ping"`
	LastVersion int64  `json:"last_version,omitempty"`
	BestNode    string `json:"best_node,omitempty" xorm:"TEXT"`
	ValidNode   string `json:"valid_node,omitempty" xorm:"TEXT"`
}

// PubConfig 需要操作的数据
type PubConfig struct {
	Remark  string `json:"remark,omitempty" xorm:"varchar(256)"`
	SubLink string `json:"sub_link,omitempty" xorm:"varchar(256)"`
}

var engine *xorm.Engine

func init() {
	var err error
	engine, err = xorm.NewEngine("sqlite3", "./config.db")
	if err != nil {
		panic(err)
	}

	if err = engine.CreateTables(new(SubConfig), new(PubConfig)); err != nil {
		panic(err)
	}
}
