package main

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"github.com/huzorro/spfactor/sexredis"
	"github.com/huzorro/woplus/tools"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"reflect"
	//	"sort"
	"strconv"
	"strings"
	"time"
)

type V1Handler struct {
	p *sexredis.RedisPool
	c *Cfg
}

type V2Handler struct {
	p *sexredis.RedisPool
	c *Cfg
}

func (self *V1Handler) SProcess(msg *sexredis.Msg) {
	log.Printf("v1 process start... %+v", msg)
	var (
		vCode string
		ok    bool
		v     VCode
		r     VResult
	)
	//msg type ok ?
	if vCode, ok = msg.Content.(string); !ok {
		log.Printf("Msg type error %+", msg)
		msg.Err = errors.New("Msg type error")
		return
	}
	//	//构造request header
	hp := url.Values{}
	hp.Add("appKey", "\""+self.c.AppKey+"\"")
	hp.Add("token", "\""+self.c.Token+"\"")

	self.c.Authorization, _ = url.QueryUnescape(strings.Replace(hp.Encode(), "&", ",", -1))
	//跳过https证书校验
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	//	certs := x509.NewCertPool()

	//	pemData, err := ioutil.ReadFile("iSimularClient.cer")
	//	if err != nil {
	//		log.Println(string(pemData))
	//	}
	//	cert, err := x509.ParseCertificate(pemData)
	//	certs.AddCert(cert)
	//	certs.AppendCertsFromPEM(pemData)
	//	tr := &http.Transport{
	//		TLSClientConfig: &tls.Config{RootCAs: certs, InsecureSkipVerify: true},
	//	}
	//	tr := &http.Transport{
	//		TLSClientConfig: &tls.Config{Certificates: []tls.Certificate{self.cert}, InsecureSkipVerify: true},
	//	}
	client := &http.Client{Transport: tr}

	req, err := http.NewRequest("POST", self.c.V1ReqUri, strings.NewReader(vCode))
	if err != nil {
		log.Printf("v1 request fails %s", err)
		msg.Err = errors.New("v1 request fails")
		return
	}

	req.Header.Set("Authorization", self.c.Authorization)
	req.Header.Set("Content-Type", self.c.ContentType)
	req.Header.Set("Accept", self.c.Accept)

	resp, err := client.Do(req)

	if err != nil {
		log.Printf("post request fails %s", err)
		msg.Err = errors.New("post request fails")
		return
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("v2 response body read fails %s", err)
		msg.Err = errors.New("v2 response body read fails")
		return
	}
	defer resp.Body.Close()

	if err != nil {
		log.Printf("v1 response fails %s", err)
		msg.Err = errors.New("v1 response fails")
		return
	}
	if err := json.Unmarshal([]byte(vCode), &v); err != nil {
		log.Printf("v1 josn Unmarshal fails %s", err)
		msg.Err = errors.New("json Unmarshal fails")
		return
	}
	if err := json.Unmarshal(body, &r); err != nil {
		log.Printf("v1 josn Unmarshal fails %s", err)
		msg.Err = errors.New("json Unmarshal fails")
		return
	}
	vr := VR{v, r}

	rc, err := self.p.Get()
	defer self.p.Close(rc)

	if err != nil {
		log.Printf("get redis connection fails %s", err)
		msg.Err = errors.New("get redis connection fails")
		return
	}
	queue := sexredis.New()
	queue.SetRClient(VR1_REQUEST_QUEUE_NAME, rc)
	js, err := json.Marshal(vr)
	log.Printf("vr1 request >> reply %s", string(js))
	if err != nil {
		log.Printf("json marshal fails %s", err)
		msg.Err = errors.New("json marshal fails")
		return
	}
	if _, err := queue.Put(js); err != nil {
		log.Printf("put vr1 request >> reply into queue fails %s", err)
		msg.Err = errors.New("put vr1 request >> reply into queue fails")
		return
	}
}

func (self *V2Handler) SProcess(msg *sexredis.Msg) {
	log.Printf("v1 process start... %+v", msg)
	var (
		vCode string
		ok    bool
		v     V2Code
		r     V2Result
	)
	//msg type ok ?
	if vCode, ok = msg.Content.(string); !ok {
		log.Printf("Msg type error %+", msg)
		msg.Err = errors.New("Msg type error")
		return
	}
	//构造request header
	hp := url.Values{}
	hp.Add("appKey", "\""+self.c.AppKey+"\"")
	hp.Add("token", "\""+self.c.Token+"\"")
	self.c.Authorization, _ = url.QueryUnescape(strings.Replace(hp.Encode(), "&", ",", -1))
	//时间戳/签名
	if err := json.Unmarshal([]byte(vCode), &v); err != nil {
		log.Printf("josn Unmarshal fails %s", err)
		msg.Err = errors.New("json Unmarshal fails")
		return
	}
	v.SignType = self.c.SignType
	v.TimeStamp = time.Now().Local().Format("20060102150405")
	sa := url.Values{}
	vType := reflect.TypeOf(&v).Elem()
	vValue := reflect.ValueOf(&v).Elem()

	for i := 0; i < vType.NumField(); i++ {
		tag := vType.Field(i).Tag.Get("json")
		fN := vType.Field(i).Name
		var value string
		if tag == "signType" || tag == "signature" {
			continue
		}
		switch vType.Field(i).Type.Kind() {
		case reflect.String:
			value = vValue.FieldByName(fN).String()
		case reflect.Int:
			value = strconv.FormatInt(vValue.FieldByName(fN).Int(), 10)
		case reflect.Float64:
			value = strconv.FormatFloat(vValue.FieldByName(fN).Float(), 'f', -1, 64)
		default:
			//
		}
		if strings.Contains(tag, "omitempty") && (value == "" || value == "0") {
			continue
		}
		sa.Add(tag, value)
	}
	unescape, _ := url.QueryUnescape(tools.Encode(sa))
	v.Signature = tools.HmacSha1(unescape, self.c.AppSecret)
	vvcode, err := json.Marshal(v)

	//	log.Println(unescape, self.c.AppSecret, string(vvcode))
	//跳过https证书校验
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, err := http.NewRequest("POST", self.c.V2ReqUri, strings.NewReader(string(vvcode)))
	if err != nil {
		log.Printf("v2 request fails %s", err)
		msg.Err = errors.New("v2 request fails")
		return
	}

	req.Header.Set("Authorization", self.c.Authorization)
	req.Header.Set("Content-Type", self.c.ContentType)
	req.Header.Set("Accept", self.c.Accept)

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("v2 request fails %s", err)
		msg.Err = errors.New("v2 request fails")
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("v2 response body read fails %s", err)
		msg.Err = errors.New("v2 response body read fails")
		return
	}
	defer resp.Body.Close()

	if err != nil {
		log.Printf("v2 response fails %s", err)
		msg.Err = errors.New("v2 response fails")
		return
	}

	if err := json.Unmarshal(body, &r); err != nil {
		log.Printf("josn Unmarshal fails %s", err)
		msg.Err = errors.New("json Unmarshal fails")
		return
	}
	vr := V2R{v, r}

	rc, err := self.p.Get()
	defer self.p.Close(rc)

	if err != nil {
		log.Printf("get redis connection fails %s", err)
		msg.Err = errors.New("get redis connection fails")
		return
	}
	queue := sexredis.New()
	queue.SetRClient(VR2_REQUEST_QUEUE_NAME, rc)
	js, err := json.Marshal(vr)
	log.Printf("vr2 request >> reply %s", string(js))
	if err != nil {
		log.Printf("json marshal fails %s", err)
		msg.Err = errors.New("json marshal fails")
		return
	}
	if _, err := queue.Put(js); err != nil {
		log.Printf("put vr2 request >> reply into queue fails %s", err)
		msg.Err = errors.New("put vr2 request >> reply into queue fails")
		return
	}
}
