// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package browscap_go

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"unicode"

	"time"

	compress "github.com/bkaradzic/go-lz4"
	cache "github.com/patrickmn/go-cache"
)

const (
	DownloadUrl     = "http://browscap.org/stream?q=PHP_BrowsCapINI"
	CheckVersionUrl = "http://browscap.org/version-number"
)

var (
	dict           *dictionary
	initialized    bool
	version        string
	debug          bool
	browscap_cache *cache.Cache
)

func Debug(val bool) {
	debug = val
}

func InitBrowsCap(path string, force bool, expiration time.Duration, cleanup time.Duration) error {
	if initialized && !force {
		return nil
	}
	var err error

	// Load ini file
	if dict, err = loadFromIniFile(path); err != nil {
		return fmt.Errorf("browscap: An error occurred while reading file, %v ", err)
	}

	browscap_cache = cache.New(expiration, cleanup)

	initialized = true
	return nil
}

func Close() {
	browscap_cache.Flush()
}

func InitializedVersion() string {
	return version
}

func LastVersion() (string, error) {
	response, err := http.Get(CheckVersionUrl)
	if err != nil {
		return "", fmt.Errorf("browscap: error sending request, %v", err)
	}
	defer response.Body.Close()

	// Get body of response
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("browscap: error reading the response data of request, %v", err)
	}

	// Check 200 status
	if response.StatusCode != 200 {
		return "", fmt.Errorf("browscap: error unexpected status code %d", response.StatusCode)
	}

	return string(body), nil
}

func DownloadFile(saveAs string) error {
	response, err := http.Get(DownloadUrl)
	if err != nil {
		return fmt.Errorf("browscap: error sending request, %v", err)
	}
	defer response.Body.Close()

	// Get body of response
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("browscap: error reading the response data of request, %v", err)
	}

	// Check 200 status
	if response.StatusCode != 200 {
		return fmt.Errorf("browscap: error unexpected status code %d", response.StatusCode)
	}

	if err = ioutil.WriteFile(saveAs, body, os.ModePerm); err != nil {
		return fmt.Errorf("browscap: error saving file, %v", err)
	}

	return nil
}

func GetBrowser(userAgent string) (browser *Browser, err error) {

	if !initialized {
		return
	}

	hash := userAgent

	browser, err = GetCache(hash)
	if err == nil {
		return
	}

	err = nil

	agent := mapToBytes(unicode.ToLower, userAgent)
	defer bytesPool.Put(agent)

	name := dict.tree.Find(agent)
	if name == "" {
		err = fmt.Errorf("Bad UA")
		return
	}

	browser = dict.getBrowser(name)
	if browser != nil {
		SetCache(userAgent, browser)
	} else {
		err = fmt.Errorf("Bad UA")
	}

	return
}

func SetCache(key string, data interface{}) (err error) {
	buf, err := Marshal(data)
	if err == nil {
		browscap_cache.SetDefault(key, buf)
	}
	return
}

func GetCache(key string) (browser *Browser, err error) {

	if buf, ok := browscap_cache.Get(key); ok {
		if jsonData, err := Unmarshal(buf.([]byte)); err == nil {
			if err = json.Unmarshal(jsonData, &browser); err != nil {
				browscap_cache.Delete(key)
			}
		} else {
			browscap_cache.Delete(key)
		}
	} else {
		err = fmt.Errorf("empty")
	}

	return
}

func Marshal(value interface{}) (bufCompress []byte, err error) {
	var (
		bufJson []byte
	)
	bufJson, err = json.Marshal(value)

	if err == nil {
		bufCompress, err = compress.Encode(nil, bufJson)
	}

	return
}

func Unmarshal(buf []byte) (value []byte, err error) {

	value, err = compress.Decode(nil, buf)

	return
}
