package main

import (
    "testing"
    "io/ioutil"
    "os"
)

func TestDetectSopsFile(t *testing.T) {

	tmp, err := ioutil.TempFile("", "utils_test.*.yaml")
	if err != nil {
		t.Errorf("Error creating temp file")
	}
	defer os.Remove(tmp.Name())

	// Test negative case
	tmp.WriteString(`---
secret: hello
`)
	res, err := DetectSopsYaml(tmp.Name())
	if res != false || err != nil {
		t.Errorf("DetectSopsYaml(tmp) = %t, %v; want false, <nil>", res, err)
	}

	// Test positive case
	tmp.Seek(0, 0)
	tmp.WriteString(`---
secret: ENC[AES256_GCM,...]
sops:
    kms: []
    gcp_kms: []
    azure_kv: []
    hc_vault: []
    lastmodified: '2020-11-03T01:45:48Z'
    pgp: []
    version: 3.6.1
`)
	res, err = DetectSopsYaml(tmp.Name())
	if res != true || err != nil {
		t.Errorf("DetectSopsYaml(tmp) = %t, %v; want true, <nil>", res, err)
	}
}
