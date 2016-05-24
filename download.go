//批量获取雅虎股票数据。
package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

const (
	UA = "Golang Downloader from Ijibu.com"

	// 每个协程下载的文件片段大小（单位：字节）
	MAX_FILE_SEGMENT_BYTES_PER_ROUTINE = 1024
)

type HttpRespBody struct {
	FileTotalSize int
}

var done_count int

func main() {

	//设置cpu的核的数量，从而实现高并发
	runtime.GOMAXPROCS(runtime.NumCPU())

	// 下载完成控制器
	var wait_group sync.WaitGroup

	var file_mutex sync.Mutex

	// 远程文件
	urls := "http://localhost/images/arich_1.jpg"

	file_info := get_file_info(urls)

	done_count = 0

	sum := 0
	// 开启协程组进行分段下载
	for offset := 0; offset < file_info.FileTotalSize; offset = offset + MAX_FILE_SEGMENT_BYTES_PER_ROUTINE {
		wait_group.Add(1)
		sum++
		fmt.Println("the sum is ", sum)
		go getShangTickerTables(urls, file_mutex, wait_group, offset, MAX_FILE_SEGMENT_BYTES_PER_ROUTINE)
	}

	// 等待下载完成
	wait_group.Wait()

	fmt.Println("main ok")
}

func get_file_info(urls string) HttpRespBody {

	// http 请求
	var req http.Request
	req.Method = "GET"
	req.Close = true

	var err error
	req.URL, err = url.Parse(urls)
	if err != nil {
		panic(err)
	}

	header := http.Header{}
	header.Set("Range", "bytes=0-1")
	header.Set("User-Agent", UA)
	req.Header = header
	resp, err := http.DefaultClient.Do(&req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	var ret HttpRespBody
	content_range := resp.Header["Content-Range"]
	ret2 := strings.Split(content_range[0], "/")
	ret3 := strings.Split(ret2[1], "]")
	ret.FileTotalSize, _ = strconv.Atoi(ret3[0])

	return ret
}

func getShangTickerTables(urls string, file_mutex sync.Mutex, wait_group sync.WaitGroup, offset int, size int) {

	// http 请求
	var req http.Request
	req.Method = "GET"
	req.Close = true

	var err error
	req.URL, err = url.Parse(urls)
	if err != nil {
		fmt.Println(err.Error())
		panic(err)
	}

	header := http.Header{}
	range_str := fmt.Sprintf("bytes=%d-%d", offset, offset+size-1)
	header.Set("Range", range_str)
	header.Set("User-Agent", UA)
	req.Header = header
	resp, err := http.DefaultClient.Do(&req)
	if err != nil {
		fmt.Println(err.Error())
		panic(err)
	}
	defer resp.Body.Close()

	file_mutex.Lock()
	defer file_mutex.Unlock()

	ff, err := os.OpenFile("d:\\44.jpg", os.O_RDWR, os.FileMode(0666))
	if err != nil {
		fmt.Println(err.Error())
		panic(err)
	}
	defer ff.Close()

	// 实际数据长度 Content-Length:[592]
	real_length, err := strconv.Atoi(resp.Header["Content-Length"][0])

	ff.Seek(int64(offset), os.SEEK_SET)
	io.CopyN(ff, resp.Body, int64(real_length))

	wait_group.Done()

	done_count++
	fmt.Println("done count is", done_count)
}
