# range_downloader

基于golang的http下载器，支持断点续传,支持http/https代理。

## USAGE

  ```
  go build main.go
  ./main -use-proxy -https
  ```

命令行选项：
  
  ```
  -http
        short for use-http
  -https
        short for use-https
  -use-http
        to use a http proxy during downloading
  -use-https
        to use a https proxy during downloading
  -use-proxy
        weather to use a proxy during downloading
  ```

## TODO

1. 支持并发下载。
2. 支持进度条提示。