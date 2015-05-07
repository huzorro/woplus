package main

import (
	"encoding/json"
	"flag"
	"github.com/go-martini/martini"
	"github.com/gosexy/redis"
	"github.com/huzorro/spfactor/sexredis"
	"github.com/huzorro/woplus/tools"
	"github.com/martini-contrib/render"
	"log"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
)

const (
	RESPONSE_OK_TEXT         = "{\"response\":\"OK\"}"
	RESPONSE_REDIS_NOK_TEXT  = "{\"response\":\"NOK\", \"text\":\"REDIS\"}"
	RESPONSE_JSON_NOK_TEXT   = "{\"response\":\"NOK\", \"text\":\"JSON\"}"
	RESPONSE_PUT_NOK_TEXT    = "{\"response\":\"NOK\", \"text\":\"PUT\"}"
	RESPONSE_GET_MO_NOK_TEXT = "{\"response\":\"NOK\", \"text\":\"GETMO\"}"
	RESPONSE_GET_MT_NOK_TEXT = "{\"response\":\"NOK\", \"text\":\"GETMT\"}"
)

type Cfg struct {
	V1ReceiveQueueName string
	V2ReceiveQueueName string
	V1RequestQueueName string
	V2RequestQueueName string
	Authorization      string
	ContentType        string
	Accept             string
	AppKey             string
	AppSecret          string
	Token              string
	V1ReqUri           string
	V2ReqUri           string
	SignType           string
}

type VCode struct {
	PaymentUser      string  `json:"paymentUser"`
	PaymentType      int     `json:"paymentType"`
	OutTradeNo       string  `json:"outTradeNo"`
	PaymentAcount    string  `json:"paymentAcount"`
	Subject          string  `json:"subject"`
	Description      string  `json:"description,omitempty"`
	Price            float64 `json:"price,omitempty"`
	Quantity         int     `json:"quantity,omitempty"`
	TotalFee         float64 `json:"totalFee"`
	ShowUrl          string  `json:"showUrl,omitempty"`
	SubscriptionType int     `json:"subscriptionType,omitempty"`
}

type V2Code struct {
	PaymentUser    string  `json:"paymentUser"`
	OutTradeNo     string  `json:"outTradeNo"`
	PaymentAcount  string  `json:"paymentAcount"`
	Subject        string  `json:"subject"`
	Description    string  `json:"description,omitempty"`
	Price          float64 `json:"price,omitempty"`
	Quantity       int     `json:"quantity,omitempty"`
	TotalFee       float64 `json:"totalFee"`
	ShowUrl        string  `json:"showUrl,omitempty"`
	Paymentcodesms int     `json:"paymentcodesms"`
	TimeStamp      string  `json:"timeStamp"`
	SignType       string  `json:"signType"`
	Signature      string  `json:"signature"`
}
type VResult struct {
	ResultCode        int    `json:"resultCode"`
	ResultDescription string `json:"resultDescription"`
}

type V2Result struct {
	ResultCode        string `json:"resultCode"`
	ResultDescription string `json:"resultDescription"`
	TransactionId     string `json:"transactionId"`
}

type VR struct {
	V VCode
	R VResult
}

type V2R struct {
	V V2Code
	R V2Result
}

func reviewV1Request(r *http.Request, w http.ResponseWriter, log *log.Logger, redisPool *sexredis.RedisPool, cfg *Cfg) (int, string) {
	w.Header().Set("Content-Type", "text/html;charset=UTF-8")
	rc, err := redisPool.Get()
	defer redisPool.Close(rc)
	if err != nil {
		log.Printf("get redis connection of pool fails %s", err)
		return http.StatusInternalServerError, RESPONSE_REDIS_NOK_TEXT
	}
	ret, _ := rc.LRange(cfg.V1RequestQueueName, 0, -1)
	rev := make([]string, 0)
	for i := len(ret) - 1; i >= 0; i-- {
		rev = append(rev, ret[i])
	}
	return http.StatusOK, strings.Join(rev, "</br>")
}

func reviewV2Request(r *http.Request, w http.ResponseWriter, log *log.Logger, redisPool *sexredis.RedisPool, cfg *Cfg) (int, string) {
	w.Header().Set("Content-Type", "text/html;charset=UTF-8")
	rc, err := redisPool.Get()
	defer redisPool.Close(rc)
	if err != nil {
		log.Printf("get redis connection of pool fails %s", err)
		return http.StatusInternalServerError, RESPONSE_REDIS_NOK_TEXT
	}
	ret, _ := rc.LRange(cfg.V2RequestQueueName, 0, -1)
	rev := make([]string, 0)
	for i := len(ret) - 1; i >= 0; i-- {
		rev = append(rev, ret[i])
	}
	return http.StatusOK, strings.Join(rev, "</br>")
}

func receiverV1(r *http.Request, w http.ResponseWriter, log *log.Logger, redisPool *sexredis.RedisPool, cfg *Cfg) (int, string) {
	r.ParseForm()
	var vCode VCode
	vType := reflect.TypeOf(&vCode).Elem()
	vValue := reflect.ValueOf(&vCode).Elem()
	for i := 0; i < vType.NumField(); i++ {
		fN := vType.Field(i).Name
		p, _ := url.QueryUnescape(r.URL.Query().Get(fN))
		switch vType.Field(i).Type.Kind() {
		case reflect.String:
			vValue.FieldByName(fN).SetString(p)
		case reflect.Int:
			in, _ := strconv.Atoi(p)
			vValue.FieldByName(fN).SetInt(int64(in))
		case reflect.Float64:
			fl, _ := strconv.ParseFloat(p, 64)
			vValue.FieldByName(fN).SetFloat(fl)
		default:
			//
		}
	}
	js, _ := json.Marshal(vCode)
	rc, err := redisPool.Get()
	defer redisPool.Close(rc)
	if err != nil {
		log.Printf("get redis connection of pool fails %s", err)
		return http.StatusInternalServerError, RESPONSE_REDIS_NOK_TEXT
	}

	queue := sexredis.New()
	queue.SetRClient(cfg.V1ReceiveQueueName, rc)
	log.Printf("receive v1 %s", string(js))

	if err != nil {
		log.Printf("json marshal fails %s", err)
		return http.StatusInternalServerError, RESPONSE_JSON_NOK_TEXT
	}
	if _, err := queue.Put(js); err != nil {
		log.Printf("put receive v1 into queue fails %s", err)
		return http.StatusInternalServerError, RESPONSE_PUT_NOK_TEXT
	}

	return http.StatusOK, RESPONSE_OK_TEXT
}

func receiverV2(r *http.Request, w http.ResponseWriter, log *log.Logger, redisPool *sexredis.RedisPool, cfg *Cfg) (int, string) {
	r.ParseForm()
	var vCode V2Code
	vType := reflect.TypeOf(&vCode).Elem()
	vValue := reflect.ValueOf(&vCode).Elem()

	for i := 0; i < vType.NumField(); i++ {
		fN := vType.Field(i).Name
		p, _ := url.QueryUnescape(r.URL.Query().Get(fN))
		switch vType.Field(i).Type.Kind() {
		case reflect.String:
			vValue.FieldByName(fN).SetString(p)
		case reflect.Int:
			in, _ := strconv.Atoi(p)
			vValue.FieldByName(fN).SetInt(int64(in))
		case reflect.Float64:
			fl, _ := strconv.ParseFloat(p, 64)
			vValue.FieldByName(fN).SetFloat(fl)
		default:
			//
		}
	}
	js, _ := json.Marshal(vCode)
	log.Printf("receive v2 %s", string(js))
	rc, err := redisPool.Get()
	defer redisPool.Close(rc)
	if err != nil {
		log.Printf("get redis connection of pool fails %s", err)
		return http.StatusInternalServerError, RESPONSE_REDIS_NOK_TEXT
	}

	queue := sexredis.New()
	queue.SetRClient(cfg.V2ReceiveQueueName, rc)

	if err != nil {
		log.Printf("json marshal fails %s", err)
		return http.StatusInternalServerError, RESPONSE_JSON_NOK_TEXT
	}
	if _, err := queue.Put(js); err != nil {
		log.Printf("put receive v1 into queue fails %s", err)
		return http.StatusInternalServerError, RESPONSE_PUT_NOK_TEXT
	}
	return http.StatusOK, RESPONSE_OK_TEXT
}

func main() {
	receiverPtr := flag.Bool("receiver", false, "v1/v2 receiver start")

	//handler msg
	v1HandlerPtr := flag.Bool("v1Handler", false, "v1 handler start")
	v2HandlerPtr := flag.Bool("v2Handler", false, "v2 handler start")

	portPtr := flag.String("port", ":10086", "service port")
	redisIdlePtr := flag.Int("redis", 20, "redis idle connections")

	//config path
	cfgPathPtr := flag.String("config", "config.json", "config path name")

	flag.Parse()

	logger := log.New(os.Stdout, "\r\n", log.Ldate|log.Ltime|log.Lshortfile)
	redisPool := &sexredis.RedisPool{make(chan *redis.Client, *redisIdlePtr), func() (*redis.Client, error) {
		client := redis.New()
		err := client.Connect("localhost", uint(6379))
		return client, err
	}}

	mtn := martini.Classic()

	mtn.Map(logger)
	mtn.Map(redisPool)
	//render
	rOptions := render.Options{}
	rOptions.Extensions = []string{".tmpl", ".html"}
	mtn.Use(render.Renderer(rOptions))

	//json config
	var cfg Cfg
	if err := tools.Json2Struct(*cfgPathPtr, &cfg); err != nil {
		log.Printf("load json config fails %s", err)
		panic(err.Error())
	}

	mtn.Map(&cfg)

	if *receiverPtr {
		mtn.Get("/receiver/v1", receiverV1)
		mtn.Get("/receiver/v2", receiverV2)
		mtn.Get("/review/v1", reviewV1Request)
		mtn.Get("/review/v2", reviewV2Request)
	}

	if *receiverPtr {
		go http.ListenAndServe(*portPtr, mtn)
	}

	if *v1HandlerPtr {
		rc, err := redisPool.Get()
		if err != nil {
			log.Printf("get redis connection fails %s", err)
			return
		}
		defer redisPool.Close(rc)
		queue := sexredis.New()
		queue.SetRClient(cfg.V1ReceiveQueueName, rc)
		queue.Worker(2, true, &V1Handler{redisPool, &cfg})
	}

	if *v2HandlerPtr {
		rc, err := redisPool.Get()
		if err != nil {
			log.Printf("get redis connection fails %s", err)
			return
		}
		defer redisPool.Close(rc)
		queue := sexredis.New()
		queue.SetRClient(cfg.V2ReceiveQueueName, rc)
		queue.Worker(2, true, &V2Handler{redisPool, &cfg})
	}
	done := make(chan bool)
	<-done
}
