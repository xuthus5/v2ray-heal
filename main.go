package main

import (
	"fmt"
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
	"v2ray-heal/config"
	"v2ray-heal/handler"
)

var confer = config.GetConfig()

func main() {
	var router = httprouter.New()
	router.GlobalOPTIONS = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Access-Control-Request-Method") != "" {
			header := w.Header()
			header.Set("Access-Control-Allow-Methods", header.Get("Allow"))
			header.Set("Access-Control-Allow-Origin", "*")
			header.Set("Access-Control-Allow-Headers", "*")
		}
		w.WriteHeader(http.StatusNoContent)
	})
	router.PanicHandler = func(w http.ResponseWriter, r *http.Request, v interface{}) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(fmt.Sprintf("err: %+v", v)))
	}

	router.GET("/sub", handler.Sub)
	router.POST("/pub", handler.Pub)

	log.Printf("%s start...", confer.ProjectName)

	if confer.Enable {
		log.Fatal(http.ListenAndServeTLS(confer.Port, confer.CrtFile, confer.KeyFile, router))
	} else {
		log.Fatal(http.ListenAndServe(confer.Port, router))
	}
}
