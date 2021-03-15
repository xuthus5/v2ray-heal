package handler

import (
	"fmt"
	"github.com/guonaihong/gout"
	"github.com/panjf2000/ants/v2"
	"github.com/remeh/sizedwaitgroup"
	"iochen.com/v2gen/v2/common/mean"
	"log"
	"psubv2ray/driver"
	"sort"
	"sync"
	"time"

	"iochen.com/v2gen/v2"
	"iochen.com/v2gen/v2/common/base64"
	"iochen.com/v2gen/v2/ping"
)

type PingInfo struct {
	Status   *ping.Status
	Duration ping.Duration
	Link     v2gen.Link
	Err      error
}

type PingInfoList []*PingInfo

func (pf *PingInfoList) Len() int {
	return len(*pf)
}

func (pf *PingInfoList) Less(i, j int) bool {
	if (*pf)[i].Err != nil {
		return false
	} else if (*pf)[j].Err != nil {
		return true
	}

	if len((*pf)[i].Status.Errors) != len((*pf)[j].Status.Errors) {
		return len((*pf)[i].Status.Errors) < len((*pf)[j].Status.Errors)
	}

	return (*pf)[i].Duration < (*pf)[j].Duration
}

func (pf *PingInfoList) Swap(i, j int) {
	(*pf)[i], (*pf)[j] = (*pf)[j], (*pf)[i]
}

const (
	pingCount  = 3
	pingDest   = "https://cloudflare.com/cdn-cgi/trace"
	tickerTime = time.Minute * 30
)

var (
	routinePool *ants.Pool
	lock        sync.RWMutex
)

func init() {
	log.SetFlags(log.Llongfile)
	var err error
	routinePool, err = ants.NewPool(100)

	if err != nil {
		log.Fatalf("new pool err: %+v", err)
	}

	// run pub daemon
	go startPub()
}

func updateSubConfig(node *driver.PubConfig, version int64) func() {
	return func() {
		log.Printf("start [%s] sub link: %s\n", node.Remark, node.SubLink)
		linkList, err := getLinkList(node.SubLink)
		if err != nil {
			log.Printf("get [%s] link node request err: unreachable\n", node.Remark)
			return
		}
		if len(linkList) == 0 {
			log.Printf("get [%s] link node empty\n", node.Remark)
			return
		}

		// 检查延迟
		pingInfoList := make(PingInfoList, len(linkList))
		wg := sizedwaitgroup.New(5)
		for i := range linkList {
			wg.Add()
			go func(i int) {
				defer func() {
					wg.Done()
				}()
				pingInfoList[i] = &PingInfo{
					Link: linkList[i],
				}
				status, err := linkList[i].Ping(pingCount, pingDest)
				if err != nil {
					log.Printf("get [%s] link status: %+v\n", node.Remark, err)
					pingInfoList[i].Err = err
					status.Durations = &ping.DurationList{-1}
					pingInfoList[i].Status = &ping.Status{
						Durations: &ping.DurationList{},
					}
				}

				if status.Durations == nil || len(*status.Durations) == 0 {
					pingInfoList[i].Err = fmt.Errorf("all error")
					status.Durations = &ping.DurationList{-1}
					pingInfoList[i].Status = &ping.Status{
						Durations: &ping.DurationList{},
					}
				} else {
					pingInfoList[i].Status = &status
				}
			}(i)
		}
		wg.Wait()

		var validNodeList PingInfoList
		for i := range pingInfoList {
			var ok bool
			pingInfoList[i].Duration, ok = mean.ArithmeticMean(pingInfoList[i].Status.Durations).(ping.Duration)
			// 延迟大于3秒小于30毫秒的 均算失败
			if !ok || int64(pingInfoList[i].Duration)/1e6 > 3000 || int64(pingInfoList[i].Duration)/1e6 <= 30 {
				pingInfoList[i].Duration = -1
			} else {
				validNodeList = append(validNodeList, pingInfoList[i])
			}
		}
		if validNodeList == nil || validNodeList.Len() == 0 {
			log.Printf("valid [%s] node empty\n", node.Remark)
			return
		}
		sort.Sort(&validNodeList)
		// 获取最优节点
		var chosenNode = validNodeList[0]
		log.Printf("get [%s] best node time: %v ms\n", node.Remark, int64(chosenNode.Duration)/1e6)
		// 比较 获取原来的数据
		lock.Lock()
		subConfig, err := driver.GetSubConfig()
		if err != nil {
			log.Printf("get [%s] err: %v", node.Remark, err)
			return
		}
		var newConfig = driver.SubConfig{
			Version: version,
			Ping:    int64(chosenNode.Duration) / 1e6,
			Master:  true,
		}
		if subConfig.Version < version || (subConfig.Version == version && subConfig.Ping > int64(chosenNode.Duration)) {
			// 满足条件 更新
			newConfig.BestNode = base64.Encode(chosenNode.Link.String())
		} else {
			// 维持以前的
			newConfig.BestNode = subConfig.BestNode
		}
		// 获取有效节点
		var validContent string
		validList := AvailableLinks(validNodeList)
		if len(validList) == 0 {
			log.Printf("[%s] valid node empty\n", node.Remark)
		} else {
			for _, config := range validList {
				validContent += config.Link.String() + "\n"
			}
			// 是否需要追加 否则直接更新
			if subConfig.Version == version && subConfig.LastVersion == version {
				t, err := base64.Decode(subConfig.ValidNode)
				if err != nil {
					log.Printf("decode [%s] valid node config err: %v", node.Remark, err)
				}
				validContent += t
			} else {
				newConfig.LastVersion = version
			}
			newConfig.ValidNode = base64.Encode(validContent)
		}
		// 更新
		err = driver.UpdateSubConfig(&newConfig)
		if err != nil {
			log.Printf("UpdateSubConfig [%s] err: %v", node.Remark, err)
			return
		}
		lock.Unlock()
	}
}

func wrap(version int64) func() {
	log.Printf("refresh at: %v", time.Now().Format("2006-01-02 15:04:05"))
	return func() {
		// 获取订阅节点列表
		configList, err := driver.GetPubConfigList()
		if err != nil {
			log.Printf("GetPubConfigList err: %v", err)
			return
		}

		for _, config := range configList {
			err = routinePool.Submit(updateSubConfig(config, version))
			if err != nil {
				log.Printf("submit routine but get err: %v", err)
				continue
			}
		}
	}
}

func startPub() {
	var ticker = time.NewTicker(tickerTime)
	defer ticker.Stop()
	for {
		if err := routinePool.Submit(wrap(time.Now().Unix())); err != nil {
			log.Printf("submit err: %v", err)
			time.Sleep(time.Second * 30)
			continue
		}
		<-ticker.C
	}
}

// 获取当前订阅的所有节点信息
func getLinkList(subLink string) ([]v2gen.Link, error) {
	var resp []byte
	err := gout.GET(subLink).SetTimeout(5 * time.Second).BindBody(&resp).Do()
	if err != nil {
		return nil, err
	}

	links, err := ParseLinks(resp)
	if err != nil {
		log.Printf("ParseLinks err: %v", err)
		return nil, err
	}

	return links, nil

}

func ParseLinks(b []byte) ([]v2gen.Link, error) {
	s, err := base64.Decode(string(b))
	if err != nil {
		log.Printf("base64.Decode err: %v", err)
		return nil, err
	}
	linkList, err := Parse(s)
	if err != nil {
		log.Printf("Parse err: %v", err)
		return nil, err
	}
	links := make([]v2gen.Link, len(linkList))
	for i := range linkList {
		links[i] = linkList[i]
	}
	return links, err
}

func AvailableLinks(pil PingInfoList) PingInfoList {
	var pingInfoList PingInfoList
	for i := range pil {
		if pil[i].Err == nil && len(pil[i].Status.Errors) == 0 &&
			int64(pil[i].Duration)/1e6 > 3 && int64(pil[i].Duration)/1e6 < 3000 {
			pingInfoList = append(pingInfoList, pil[i])
		}
	}

	return pingInfoList
}
