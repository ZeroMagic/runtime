// Copyright (c) 2017 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0
//

package virtcontainers

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/kata-containers/runtime/virtcontainers/store"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var testDefaultLogger = logrus.WithField("proxy", "test")

func testSetProxyType(t *testing.T, value string, expected ProxyType) {
	var proxyType ProxyType

	err := (&proxyType).Set(value)
	if err != nil {
		t.Fatal(err)
	}

	if proxyType != expected {
		t.Fatalf("Got %s\nExpecting %s", proxyType, expected)
	}
}

func TestSetKataProxyType(t *testing.T) {
	testSetProxyType(t, "kataProxy", KataProxyType)
}

func TestSetNoopProxyType(t *testing.T) {
	testSetProxyType(t, "noopProxy", NoopProxyType)
}

func TestSetNoProxyType(t *testing.T) {
	testSetProxyType(t, "noProxy", NoProxyType)
}

func TestSetKataBuiltInProxyType(t *testing.T) {
	testSetProxyType(t, "kataBuiltInProxy", KataBuiltInProxyType)
}

func TestSetUnknownProxyType(t *testing.T) {
	var proxyType ProxyType

	unknownType := "unknown"

	err := (&proxyType).Set(unknownType)
	if err == nil {
		t.Fatalf("Should fail because %s type used", unknownType)
	}

	if proxyType == NoopProxyType ||
		proxyType == NoProxyType ||
		proxyType == KataProxyType {
		t.Fatalf("%s proxy type was not expected", proxyType)
	}
}

func testStringFromProxyType(t *testing.T, proxyType ProxyType, expected string) {
	proxyTypeStr := (&proxyType).String()
	if proxyTypeStr != expected {
		t.Fatalf("Got %s\nExpecting %s", proxyTypeStr, expected)
	}
}

func TestStringFromKataProxyType(t *testing.T) {
	proxyType := KataProxyType
	testStringFromProxyType(t, proxyType, "kataProxy")
}

func TestStringFromNoProxyType(t *testing.T) {
	proxyType := NoProxyType
	testStringFromProxyType(t, proxyType, "noProxy")
}

func TestStringFromNoopProxyType(t *testing.T) {
	proxyType := NoopProxyType
	testStringFromProxyType(t, proxyType, "noopProxy")
}

func TestStringFromKataBuiltInProxyType(t *testing.T) {
	proxyType := KataBuiltInProxyType
	testStringFromProxyType(t, proxyType, "kataBuiltInProxy")
}

func TestStringFromUnknownProxyType(t *testing.T) {
	var proxyType ProxyType
	testStringFromProxyType(t, proxyType, "")
}

func testNewProxyFromProxyType(t *testing.T, proxyType ProxyType, expected proxy) {
	result, err := newProxy(proxyType)
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(result, expected) == false {
		t.Fatalf("Got %+v\nExpecting %+v", result, expected)
	}
}

func TestNewProxyFromKataProxyType(t *testing.T) {
	proxyType := KataProxyType
	expectedProxy := &kataProxy{}
	testNewProxyFromProxyType(t, proxyType, expectedProxy)
}

func TestNewProxyFromNoProxyType(t *testing.T) {
	proxyType := NoProxyType
	expectedProxy := &noProxy{}
	testNewProxyFromProxyType(t, proxyType, expectedProxy)
}

func TestNewProxyFromNoopProxyType(t *testing.T) {
	proxyType := NoopProxyType
	expectedProxy := &noopProxy{}
	testNewProxyFromProxyType(t, proxyType, expectedProxy)
}

func TestNewProxyFromKataBuiltInProxyType(t *testing.T) {
	proxyType := KataBuiltInProxyType
	expectedProxy := &kataBuiltInProxy{}
	testNewProxyFromProxyType(t, proxyType, expectedProxy)
}

func TestNewProxyFromUnknownProxyType(t *testing.T) {
	var proxyType ProxyType

	_, err := newProxy(proxyType)
	if err != nil {
		t.Fatal(err)
	}
}

func testNewProxyFromSandboxConfig(t *testing.T, sandboxConfig SandboxConfig) {
	if _, err := newProxy(sandboxConfig.ProxyType); err != nil {
		t.Fatal(err)
	}

	if err := validateProxyConfig(sandboxConfig.ProxyConfig); err != nil {
		t.Fatal(err)
	}

}

var testProxyPath = "proxy-path"

func TestNewProxyConfigFromKataProxySandboxConfig(t *testing.T) {
	proxyConfig := ProxyConfig{
		Path: testProxyPath,
	}

	sandboxConfig := SandboxConfig{
		ProxyType:   KataProxyType,
		ProxyConfig: proxyConfig,
	}

	testNewProxyFromSandboxConfig(t, sandboxConfig)
}

func TestNewProxyConfigNoPathFailure(t *testing.T) {
	if err := validateProxyConfig(ProxyConfig{}); err == nil {
		t.Fatal("Should fail because ProxyConfig has no Path")
	}
}

const sandboxID = "123456789"

func testDefaultProxyURL(expectedURL string, socketType string, sandboxID string) error {
	sandbox := &Sandbox{
		id: sandboxID,
	}

	url, err := defaultProxyURL(sandbox.id, socketType)
	if err != nil {
		return err
	}

	if url != expectedURL {
		return fmt.Errorf("Mismatched URL: %s vs %s", url, expectedURL)
	}

	return nil
}

func TestDefaultProxyURLUnix(t *testing.T) {
	path := filepath.Join(store.SandboxRuntimeRootPath(sandboxID), "proxy.sock")
	socketPath := fmt.Sprintf("unix://%s", path)

	if err := testDefaultProxyURL(socketPath, SocketTypeUNIX, sandboxID); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultProxyURLVSock(t *testing.T) {
	if err := testDefaultProxyURL("", SocketTypeVSOCK, sandboxID); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultProxyURLUnknown(t *testing.T) {
	path := filepath.Join(store.SandboxRuntimeRootPath(sandboxID), "proxy.sock")
	socketPath := fmt.Sprintf("unix://%s", path)

	if err := testDefaultProxyURL(socketPath, "foobar", sandboxID); err == nil {
		t.Fatal()
	}
}

func testProxyStart(t *testing.T, agent agent, proxy proxy) {
	assert := assert.New(t)

	assert.NotNil(proxy)

	tmpdir, err := ioutil.TempDir("", "")
	assert.NoError(err)
	defer os.RemoveAll(tmpdir)

	type testData struct {
		params      proxyParams
		expectedURI string
		expectError bool
	}

	invalidPath := filepath.Join(tmpdir, "enoent")
	expectedSocketPath := filepath.Join(store.SandboxRuntimeRootPath(testSandboxID), "proxy.sock")
	expectedURI := fmt.Sprintf("unix://%s", expectedSocketPath)

	data := []testData{
		{proxyParams{}, "", true},
		{
			// no path
			proxyParams{
				id:         "foobar",
				agentURL:   "agentURL",
				consoleURL: "consoleURL",
				logger:     testDefaultLogger,
			},
			"", true,
		},
		{
			// invalid path
			proxyParams{
				id:         "foobar",
				path:       invalidPath,
				agentURL:   "agentURL",
				consoleURL: "consoleURL",
				logger:     testDefaultLogger,
			},
			"", true,
		},
		{
			// good case
			proxyParams{
				id:         testSandboxID,
				path:       "echo",
				agentURL:   "agentURL",
				consoleURL: "consoleURL",
				logger:     testDefaultLogger,
			},
			expectedURI, false,
		},
	}

	for _, d := range data {
		pid, uri, err := proxy.start(d.params)
		if d.expectError {
			assert.Error(err)
			continue
		}

		assert.NoError(err)
		assert.True(pid > 0)
		assert.Equal(d.expectedURI, uri)
	}
}

func TestValidateProxyConfig(t *testing.T) {
	assert := assert.New(t)

	config := ProxyConfig{}
	err := validateProxyConfig(config)
	assert.Error(err)

	config.Path = "foobar"
	err = validateProxyConfig(config)
	assert.Nil(err)
}

func TestValidateProxyParams(t *testing.T) {
	assert := assert.New(t)

	p := proxyParams{}
	err := validateProxyParams(p)
	assert.Error(err)

	p.path = "foobar"
	err = validateProxyParams(p)
	assert.Error(err)

	p.id = "foobar1"
	err = validateProxyParams(p)
	assert.Error(err)

	p.agentURL = "foobar2"
	err = validateProxyParams(p)
	assert.Error(err)

	p.consoleURL = "foobar3"
	err = validateProxyParams(p)
	assert.Error(err)

	p.logger = &logrus.Entry{}
	err = validateProxyParams(p)
	assert.Nil(err)
}
