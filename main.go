package main

import (
	"io/ioutil"
	"net/http"
	"fmt"
	"strconv"
	"os"
	"io"
	"net/url"
	"strings"
	"encoding/json"
	"flag"
)

const chunkSize = 1024 
var config *Config
var useProxy = flag.Bool("use-proxy", false, "weather to use a proxy during downloading")
var useHTTP = flag.Bool("use-http", false, "to use a http proxy during downloading")
var useHTTPS = flag.Bool("use-https", false, "to use a https proxy during downloading")

func init() {
	flag.BoolVar(useHTTP, "http", false, "short for use-http")
	flag.BoolVar(useHTTPS, "https", false, "short for use-https")
	config = getConfig("./config.json")
}

type Config struct {
	HTTPProxy string `json:"http"`
	HTTPSProxy string `json:"https"`
}

func getConfig(filePath string) *Config {
	raw, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	var c Config
	json.Unmarshal(raw, &c)
	return &c
}

func httpGet(url string, start, end interface{}) (*http.Response, error) {
	client := &http.Client{ }
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	switch start.(type) {
	case int:
		switch end.(type) {
		case int:
			req.Header.Add("Range", fmt.Sprintf("bytes=%s-%s", strconv.Itoa(start.(int)), strconv.Itoa(end.(int))))
		case bool:
			req.Header.Add("Range", fmt.Sprintf("bytes=%s-", strconv.Itoa(start.(int))))
		}
	case bool:
		switch end.(type) {
		case int:
		 req.Header.Add("Range", fmt.Sprintf("bytes=-%s", strconv.Itoa(end.(int))))
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func httpRangeHead(url string) (*http.Response, error){
	client := &http.Client{}
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Range", "bytes=0-")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Find a nice func name
func supportRangeTransfer(url string) (length int64, err error) {
	resp, err := httpRangeHead(url)
	if err != nil {
		return 
	}

	switch resp.StatusCode {
	case http.StatusPartialContent:
		contentLength := resp.Header.Get("Content-Length")
		length, err = strconv.ParseInt(contentLength, 10, 0)
	default:
		length = 0
	}
	return
}

func downLoad(url, fileName string) error {
	var chunks, remLength int
	length, err := supportRangeTransfer(url)
	if err != nil {
		return err
	}
	fileInfo, err := os.Stat(fileName)
	if length > 0 {
		if os.IsNotExist(err) {
			chunks, remLength = splitLength(int(length))
		} else {
			chunks, remLength = splitLength(int(length-fileInfo.Size()))
		}
		for i := 0; i < chunks; i++ {
			err := downLoadFileChunk(url, i * chunkSize, (i+1)*chunkSize, fileName)
			if err != nil {
				return err
			}
		}
		err := downLoadFileChunk(url, false, remLength, fileName)
		if err != nil {
			return err
		}
	} else {
		err := downLoadFile(url, fileName)
		if err != nil {
			return err
	    }
	}
	return nil
}

func downLoadFile(url, fileName string) error {
    var file *os.File
	_, err := os.Stat(fileName)
	if !os.IsNotExist(err) {
		err = os.Remove(fileName)
		if err != nil {
			return err
		}
	}
	file, err = os.Create(fileName)
	defer file.Close()
	if err != nil {
		return err
	}
	resp, err := httpGet(url, false, false)
	if err != nil {
		return err
	}
	_, err = io.Copy(file, resp.Body)
	return err
}

func downLoadFileChunk(url string, start, end interface{}, fileName string) (err error) {
	var file *os.File
	var resp *http.Response
	var bytes []byte
	_, err = os.Stat(fileName)
	resp, err = httpGet(url, start, end)
	if err != nil {
		return
	}

	if os.IsNotExist(err) {
		file, err = os.Create(fileName)
		if err != nil {
			return 
		}
		io.Copy(file, resp.Body)
	} else {
		file, err = os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return
		}
		bytes, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return
		}
		if _, err = file.Write(bytes); err != nil {
			return
		}
	}
	defer file.Close()
	return
}

func checkValidURL(uri string) bool {
	_, err := url.ParseRequestURI(uri)
	if err != nil {
		return false
	}
	return true
}

func getFileName(uri string) string {
	tokens := strings.Split(uri, "/")
	fileName := tokens[len(tokens)-1]

	if fileName == "" {
		fileName = "default.file"
	}
	return fileName
}

func splitLength(length int) (int, int) {
	if length < chunkSize {
		return length, 0
	} 
	chunks := length / chunkSize
	remLength := length % chunkSize
	return chunks, remLength
}

func main() {
	flag.Parse()
	if *useProxy {
		var urlString string

		if *useHTTPS {
			urlString = config.HTTPSProxy
		} else if *useHTTP {
			urlString = config.HTTPProxy
		} else {
			urlString = config.HTTPSProxy
		}

		proxyURL, err := url.Parse(urlString)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}

		http.DefaultTransport = &http.Transport{Proxy: http.ProxyURL(proxyURL)}
	}
	
	for {
		var bytes []byte
		var fileName string
		for i := 0; i < 6; i++ {
			fmt.Print("Enter url: ")
			if i == 5 {
				fmt.Println("You entered invalid url too many times! exit!")
			}
			if _, err := fmt.Scan(&bytes); err == nil {
				if checkValidURL(string(bytes)) {
					fileName = getFileName(string(bytes))
					err := downLoad(string(bytes), fileName)
					if err != nil {
						fmt.Println(err)
						fmt.Println("Error occured while downloading!")

						os.Exit(-1)
					}
				} else {
					fmt.Println("Invalid url, please enter an valid url!")
				}
			}
		}
	}
}