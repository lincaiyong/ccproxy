package main

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/lincaiyong/log"
	"github.com/lincaiyong/uniapi/service/monica"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"
)

func extractToolUse(s string) [][2]string {
	items := regexp.MustCompile(`(?s)<use tool="(.+?)">(.+?)</use>`).FindAllStringSubmatch(s, -1)
	ret := make([][2]string, 0)
	for _, item := range items {
		args := strings.TrimSpace(item[2])
		ret = append(ret, [2]string{item[1], args})
	}
	return ret
}

func handler(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			log.ErrorLog("unexpected error: %v", err)
			c.String(http.StatusInternalServerError, fmt.Sprintf("%v", err))
		}
	}()

	var req Request
	err := c.ShouldBindJSON(&req)
	if err != nil {
		var r map[string]interface{}
		_ = c.ShouldBindJSON(&r)
		log.ErrorLog("invalid request: %v", err)
		c.String(http.StatusBadRequest, fmt.Sprintf("bad request: %v", err))
		return
	}
	// req.Thinking "thinking": { "budget_tokens": 4000, "type": "enabled" }
	if req.ToolChoice != nil || req.StopSequences != nil {
		b, _ := json.MarshalIndent(req, "", "\t")
		log.ErrorLog("not implemented features in request: %s", string(b))
		c.String(http.StatusBadRequest, "request contains unsupported features")
		return
	}
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Expose-Headers", "Content-Type")

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	var sb strings.Builder
	q := req.Compose()
	if i := strings.Index(q, "</tools>"); i != -1 {
		log.InfoLog("req: %s", q[i+8:])
	} else {
		log.InfoLog("req: %s", q)
	}

	resp := Response{}
	resp.Write(c, NewMessageStartEvent())
	resp.Write(c, NewContentBlockStartEvent())
	resp.Write(c, NewPingEvent())
	log.InfoLog("model: %s", req.Model)
	monica.Init(os.Getenv("MONICA_SESSION_ID"))
	_, err = monica.ChatCompletion(req.Model, q, func(s string) {
		sb.WriteString(s)
		resp.Write(c, NewContentBlockDeltaEvent(s))
	})

	if err != nil {
		log.ErrorLog("failed to chat completion: %v", err)
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	resp.Write(c, NewContentBlockStopEvent())
	resp.Write(c, NewMessageStopEvent())

	answer := sb.String()
	log.InfoLog("resp: %s", answer)

	tools := extractToolUse(answer)
	for _, tool := range tools {
		resp.IncIndex()
		name, args := tool[0], tool[1]
		resp.Write(c, NewContentBlockStartEventWithTool(name))
		resp.Write(c, NewContentBlockDeltaEventWithTool(name, args))
		resp.Write(c, NewContentBlockStopEventWithTool(name))
		resp.Write(c, NewMessageDeltaEventWithTool(name))
		resp.Write(c, NewMessageStopEvent())
	}
	_, err = c.Writer.Write([]byte("data: [DONE]"))
	if err != nil {
		log.ErrorLog("fail to write: %v", err)
	}
	c.Writer.Flush()

	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.String(http.StatusOK, "ok")
}

func main() {
	port := 9123
	logPath := "/tmp/ccproxy.log"
	if err := log.SetLogPath(logPath); err != nil {
		log.ErrorLog("fail to set log file path: %v", err)
		os.Exit(1)
	}
	log.InfoLog("cmd line: %s", strings.Join(os.Args, " "))
	log.InfoLog("log path: %v", logPath)
	log.InfoLog("port: %d", port)
	log.InfoLog("pid: %d", os.Getpid())
	wd, _ := os.Getwd()
	log.InfoLog("work dir: %s", wd)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		log.InfoLog("receive quit signal")
		os.Exit(0)
	}()

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		start := time.Now()
		log.InfoLog(" %s | %s", c.Request.URL.Path, c.ClientIP())
		c.Next()
		log.InfoLog(" %s | %s | %v | %d", c.Request.URL.Path, c.ClientIP(), time.Since(start), c.Writer.Status())
	})
	router.POST("/v1/messages", handler)

	log.InfoLog("starting server at 127.0.0.1:%d", port)
	err := router.Run(fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		log.ErrorLog("fail to run http server: %v", err)
		os.Exit(1)
	}
}
