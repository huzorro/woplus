package tools

import (
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
)

func Json2Struct(path string, c interface{}) error {
	bytes, err := ioutil.ReadFile(path)

	if err != nil {
		return err
	}

	if err := json.Unmarshal(bytes, c); err != nil {
		return err
	}
	return nil
}

func Cert(path string) tls.Certificate {
	cert2_b, _ := ioutil.ReadFile(path)

	cert := tls.Certificate{
		Certificate: [][]byte{cert2_b},
	}
	return cert
}
