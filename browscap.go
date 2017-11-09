// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package browscap_go

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"unicode"

	"time"

	cache2 "github.com/patrickmn/go-cache"
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
	browscap_cache *cache2.Cache
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

	browscap_cache = cache2.New(expiration, cleanup)

	initialized = true
	return nil
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

func GetBrowser(userAgent string) (browser *Browser, ok bool) {
	if !initialized {
		return
	}

	if cache, haveCache := browscap_cache.Get(userAgent); haveCache {
		if browser, ok = cache.(*Browser); ok {
			return
		} else {
			browscap_cache.Delete(userAgent)
		}
	}

	agent := mapToBytes(unicode.ToLower, userAgent)
	defer bytesPool.Put(agent)

	name := dict.tree.Find(agent)
	if name == "" {
		return
	}

	browser = dict.getBrowser(name)
	if browser != nil {
		ok = true
		browscap_cache.SetDefault(userAgent, browser)
	}

	return
}
