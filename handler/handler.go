package handler

import (
	"encoding/json"
	"github.com/julienschmidt/httprouter"
	"io/ioutil"
	"log"
	"net/http"
	"psubv2ray/driver"
	"strconv"
	"time"
)

// ResponseWrite 输出返回结果
func ResponseWrite(w http.ResponseWriter, response []byte, code int) {
	//公共的响应头设置
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(string(response))))

	if code == 0 {
		code = http.StatusOK
	}
	w.WriteHeader(code)
	_, _ = w.Write(response)
	return
}

func Sub(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var isBest = r.URL.Query().Get("best")
	var content string

	subConfig, err := driver.GetSubConfig()
	if err != nil {
		log.Printf("GetSubConfig err: %+v", err)
		ResponseWrite(w, []byte(err.Error()), http.StatusInternalServerError)
		return
	}

	if isBest != "" {
		content = subConfig.BestNode
	} else {
		content = subConfig.ValidNode
	}
	ResponseWrite(w, []byte(content), http.StatusOK)
}

func Pub(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var pubConfig driver.PubConfig

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("read body err: %+v", err)
		ResponseWrite(w, []byte(err.Error()), http.StatusBadRequest)
		return
	}

	err = json.Unmarshal(body, &pubConfig)
	if err != nil {
		log.Printf("Unmarshal body err: %+v", err)
		ResponseWrite(w, []byte(err.Error()), http.StatusBadRequest)
		return
	}

	err = driver.AddPubConfig(&pubConfig)
	if err != nil {
		log.Printf("AddPubConfig err: %+v", err)
		ResponseWrite(w, []byte(err.Error()), http.StatusInternalServerError)
		return
	}

	// 立即执行一次
	err = routinePool.Submit(wrap(time.Now().Unix()))
	if err != nil {
		log.Printf("Submit err: %+v", err)
	}

	ResponseWrite(w, []byte("ok"), http.StatusOK)
}
