package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"iochen.com/v2gen/v2/common/base64"
	"iochen.com/v2gen/v2/common/split"
	"iochen.com/v2gen/v2/ping"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
	"unicode"
	"v2ray.com/core"
	"v2ray.com/core/app/dispatcher"
	applog "v2ray.com/core/app/log"
	"v2ray.com/core/app/proxyman"
	commlog "v2ray.com/core/common/log"
	v2net "v2ray.com/core/common/net"
	"v2ray.com/core/common/serial"
	"v2ray.com/core/infra/conf"

	_ "v2ray.com/core/app/dispatcher"
	_ "v2ray.com/core/app/proxyman/inbound"
	_ "v2ray.com/core/app/proxyman/outbound"
	_ "v2ray.com/core/proxy/vmess/outbound"

	_ "v2ray.com/core/transport/internet/http"
	_ "v2ray.com/core/transport/internet/kcp"
	_ "v2ray.com/core/transport/internet/quic"
	_ "v2ray.com/core/transport/internet/tcp"
	_ "v2ray.com/core/transport/internet/tls"
	_ "v2ray.com/core/transport/internet/udp"
	_ "v2ray.com/core/transport/internet/websocket"

	_ "v2ray.com/core/transport/internet/headers/http"
	_ "v2ray.com/core/transport/internet/headers/noop"
	_ "v2ray.com/core/transport/internet/headers/srtp"
	_ "v2ray.com/core/transport/internet/headers/tls"
	_ "v2ray.com/core/transport/internet/headers/utp"
	_ "v2ray.com/core/transport/internet/headers/wechat"
	_ "v2ray.com/core/transport/internet/headers/wireguard"
)

type LinkV2 struct {
	Ps         string      `json:"ps"`
	Add        string      `json:"add"`
	Port       interface{} `json:"port"`
	ID         string      `json:"id"`
	Aid        interface{} `json:"aid"`
	Net        string      `json:"net"`
	Type       string      `json:"type"`
	Host       string      `json:"host"`
	Path       string      `json:"path"`
	TLS        string      `json:"tls"`
	Version    interface{} `json:"v"`
	VerifyCert bool        `json:"verify_cert"`
	HeaderType string      `json:"headerType"`
	Remark     string      `json:"remark"`
}

var (
	ErrWrongProtocol = errors.New("wrong protocol")
)

func ParseSingle(vmessURL string) (*LinkV2, error) {
	if len(vmessURL) < 8 {
		return &LinkV2{}, errors.New(fmt.Sprint("wrong url:", vmessURL))
	}
	if vmessURL[:8] != "vmess://" {
		return &LinkV2{}, ErrWrongProtocol
	}

	j, err := base64.Decode(vmessURL[8:])
	if err != nil {
		log.Printf("base64.Decode err: %+v", err)
		return &LinkV2{}, err
	}

	node := &LinkV2{}
	node.Version = json.RawMessage("2")
	err = json.Unmarshal([]byte(j), node)
	if err != nil {
		log.Printf("json.Unmarshal err: %+v\ncontent: %+v", err, j)
		return node, err
	}
	return node, nil
}

func Parse(s string) ([]*LinkV2, error) {
	var vl []*LinkV2
	urlList := split.Split(s)
	for i := 0; i < len(urlList); i++ {
		node, err := ParseSingle(urlList[i])
		if err != nil {
			if err == ErrWrongProtocol {
				continue
			} else {
				return nil, err
			}
		}
		vl = append(vl, node)
	}
	return vl, nil
}

func (node *LinkV2) Config() map[string]string {
	var config = make(map[string]string)
	// set node settings
	config["address"] = node.Add
	config["serverPort"] = fmt.Sprintf("%v", node.Port)
	config["uuid"] = node.ID
	config["aid"] = fmt.Sprintf("%v", node.Aid)
	config["streamSecurity"] = node.TLS
	config["network"] = node.Net
	config["tls"] = node.TLS
	config["type"] = node.Type
	config["host"] = node.Host
	config["type"] = node.Type
	config["path"] = node.Path
	config["version"] = "2"
	return config
}

func (node *LinkV2) String() string {
	b, _ := json.Marshal(node)
	return "vmess://" + base64.Encode(string(b))
}

func redact(str string) string {
	var result []rune
	for _, v := range str {
		if unicode.IsDigit(v) {
			result = append(result, '0')
			continue
		}

		if unicode.IsUpper(v) {
			result = append(result, 'X')
			continue
		}

		if unicode.IsLower(v) {
			result = append(result, 'x')
			continue
		}

		result = append(result, v)
	}
	return string(result)
}

func (node *LinkV2) Safe() string {
	safeLinkV2 := &LinkV2{
		Ps:         node.Ps,
		Add:        redact(node.Add),
		Port:       node.Port,
		ID:         redact(node.ID),
		Aid:        node.Aid,
		Net:        node.Net,
		Type:       node.Type,
		Host:       redact(node.Host),
		Path:       redact(node.Path),
		Version:    node.Version,
		VerifyCert: node.VerifyCert,
		Remark:     node.Ps,
		TLS:        node.TLS,
	}
	b, _ := json.Marshal(safeLinkV2)
	return string(b)
}

func (node *LinkV2) DestAddr() string {
	return node.Add
}

func (node *LinkV2) Description() string {
	return node.Ps
}

func (node *LinkV2) Ping(round int, dst string) (ping.Status, error) {
	server, err := startV2Ray(node, false, false)
	if err != nil {
		return ping.Status{}, err
	}

	defer func() {
		if err := server.Close(); err != nil {
			log.Println(err)
		}
	}()

	ps := ping.Status{
		Durations: &ping.DurationList{},
	}

	timeout := make(chan bool, 1)

	go func() {
		time.Sleep(3 * time.Duration(round) * time.Second)
		timeout <- true
	}()

L:
	for count := 0; count < round; count++ {
		chDelay := make(chan time.Duration)
		go func() {
			delay, err := measureDelay(server, 3*time.Second, dst)
			if err != nil {
				ps.Errors = append(ps.Errors, &err)
			}
			chDelay <- delay
		}()

		select {
		case delay := <-chDelay:
			if delay > 0 {
				*ps.Durations = append(*ps.Durations, ping.Duration(delay))
			}
		case <-timeout:
			break L
		}
	}
	return ps, nil
}

func Vmess2Outbound(v *LinkV2, usemux bool) (*core.OutboundHandlerConfig, error) {
	out := &conf.OutboundDetourConfig{}
	out.Tag = "proxy"
	out.Protocol = "vmess"
	out.MuxSettings = &conf.MuxConfig{}
	if usemux {
		out.MuxSettings.Enabled = true
		out.MuxSettings.Concurrency = 8
	}

	p := conf.TransportProtocol(v.Net)
	s := &conf.StreamConfig{
		Network:  &p,
		Security: v.TLS,
	}

	switch v.Net {
	case "tcp":
		s.TCPSettings = &conf.TCPConfig{}
		if v.Type == "" || v.Type == "none" {
			s.TCPSettings.HeaderConfig = json.RawMessage([]byte(`{ "type": "none" }`))
		} else {
			pathb, _ := json.Marshal(strings.Split(v.Path, ","))
			hostb, _ := json.Marshal(strings.Split(v.Host, ","))
			s.TCPSettings.HeaderConfig = json.RawMessage([]byte(fmt.Sprintf(`
			{
				"type": "http",
				"request": {
					"path": %s,
					"headers": {
						"Host": %s
					}
				}
			}
			`, string(pathb), string(hostb))))
		}
	case "kcp":
		s.KCPSettings = &conf.KCPConfig{}
		s.KCPSettings.HeaderConfig = json.RawMessage([]byte(fmt.Sprintf(`{ "type": "%s" }`, v.Type)))
	case "ws":
		s.WSSettings = &conf.WebSocketConfig{}
		s.WSSettings.Path = v.Path
		s.WSSettings.Headers = map[string]string{
			"Host": v.Host,
		}
	case "h2", "http":
		s.HTTPSettings = &conf.HTTPConfig{
			Path: v.Path,
		}
		if v.Host != "" {
			h := conf.StringList(strings.Split(v.Host, ","))
			s.HTTPSettings.Host = &h
		}
	}

	if v.TLS == "tls" {
		s.TLSSettings = &conf.TLSConfig{
			Insecure: true,
		}
		if v.Host != "" {
			s.TLSSettings.ServerName = v.Host
		}
	}

	out.StreamSetting = s
	oset := json.RawMessage([]byte(fmt.Sprintf(`{
  "vnext": [
    {
      "address": "%s",
      "port": %v,
      "users": [
        {
          "id": "%s",
          "alterId": %v,
          "security": "auto"
        }
      ]
    }
  ]
}`, v.Add, v.Port, v.ID, v.Aid)))
	out.Settings = &oset
	return out.Build()
}

func startV2Ray(node *LinkV2, verbose, usemux bool) (*core.Instance, error) {
	loglevel := commlog.Severity_Error
	if verbose {
		loglevel = commlog.Severity_Debug
	}

	ob, err := Vmess2Outbound(node, usemux)
	if err != nil {
		return nil, err
	}
	config := &core.Config{
		App: []*serial.TypedMessage{
			serial.ToTypedMessage(&applog.Config{
				ErrorLogType:  applog.LogType_Console,
				ErrorLogLevel: loglevel,
			}),
			serial.ToTypedMessage(&dispatcher.Config{}),
			serial.ToTypedMessage(&proxyman.InboundConfig{}),
			serial.ToTypedMessage(&proxyman.OutboundConfig{}),
		},
	}

	// commlog.RegisterHandler(commlog.NewLogger(commlog.CreateStderrLogWriter()))
	config.Outbound = []*core.OutboundHandlerConfig{ob}
	server, err := core.New(config)
	if err != nil {
		return nil, err
	}

	return server, nil
}

func measureDelay(inst *core.Instance, timeout time.Duration, dest string) (time.Duration, error) {
	start := time.Now()
	code, _, err := CoreHTTPRequest(inst, timeout, "GET", dest)
	if err != nil {
		return -1, err
	}
	if code > 399 {
		return -1, fmt.Errorf("status incorrect (>= 400): %d", code)
	}
	return time.Since(start), nil
}

func CoreHTTPClient(inst *core.Instance, timeout time.Duration) (*http.Client, error) {

	if inst == nil {
		return nil, errors.New("core instance nil")
	}

	tr := &http.Transport{
		DisableKeepAlives: true,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			dest, err := v2net.ParseDestination(fmt.Sprintf("%s:%s", network, addr))
			if err != nil {
				return nil, err
			}
			return core.Dial(ctx, inst, dest)
		},
	}

	c := &http.Client{
		Transport: tr,
		Timeout:   timeout,
	}

	return c, nil
}

func CoreHTTPRequest(inst *core.Instance, timeout time.Duration, method, dest string) (int, []byte, error) {

	c, err := CoreHTTPClient(inst, timeout)
	if err != nil {
		return 0, nil, err
	}

	req, _ := http.NewRequest(method, dest, nil)
	resp, err := c.Do(req)
	if err != nil {
		return -1, nil, err
	}
	defer resp.Body.Close()

	b, _ := ioutil.ReadAll(resp.Body)
	return resp.StatusCode, b, nil
}
