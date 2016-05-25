package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
)

/*
	请求下载
	@param remote_url 下载的资源
	@param save_filename 保存的文件
*/
func DownLoad(remote_url string, save_filename string) error {

	//设置cpu的核的数量，从而实现高并发
	runtime.GOMAXPROCS(runtime.NumCPU())

	// 创建下载会话
	var session DownloadSession
	session.remote_url = remote_url
	session.save_filename = save_filename

	err := init_session(&session)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	defer func() {
		if session.save_file != nil {
			session.save_file.Close()
			session.save_file = nil
		}

		if session.tdg_file != nil {
			session.tdg_file.Close()
			session.tdg_file = nil
		}
	}()

	// 开启协程组进行分段下载
	for segment_idx := 0; segment_idx*int(session.tdg_info.FileSegmentSize) < int(session.tdg_info.TotalSize); segment_idx++ {
		// 检测是否已经下载
		if !session.tdg_info.IsSegmentDownloaded(segment_idx) {
			session.wait_group.Add(1)

			go getShangTickerTables(&session, segment_idx)
		}
	}

	// 等待下载完成
	session.wait_group.Wait()

	fmt.Println("download complete.")
	// fmt.Println(session.tdg_info)

	return nil
}

func init_session(session *DownloadSession) (err error) {
	defer func() {
		if err != nil {

			err = errors.New("init_session() error : " + err.Error())

			if session.save_file != nil {
				session.save_file.Close()
				session.save_file = nil
			}

			if session.tdg_file != nil {
				session.tdg_file.Close()
				session.tdg_file = nil
			}
		}
	}()

	// 获取远程文件信息
	remote_file_info := get_remote_res_info(session.remote_url)

	// 检测下载文件的状态
	err = check_save_file(session, remote_file_info.FileTotalSize, MAX_FILE_SEGMENT_BYTES_PER_ROUTINE)
	if err != nil {
		return err
	}

	// 实际存储文件
	save_file, err := os.OpenFile(session.save_filename, os.O_RDWR, os.FileMode(0666))
	if err != nil {
		fmt.Println(err.Error())
		panic(err)
	}

	// 实际存储文件的摘要信息
	tdg_file, err := os.OpenFile(session.save_filename+".tdg", os.O_RDWR, os.FileMode(0666))
	if err != nil {
		fmt.Println(err.Error())
		panic(err)
	}

	session.save_file = save_file
	session.tdg_file = tdg_file

	// 读取 tdg 信息
	err = read_tdg_info(session)
	if err != nil {
		return errors.New("read_tdg_info failed:" + err.Error())
	}

	return nil
}

func check_save_file(session *DownloadSession, file_total_size int, file_segment_size int) error {

	// 检测本地现在文件状态, 判断不存在则创建
	_, err := os.Stat(session.save_filename)
	if err != nil || !os.IsExist(err) {
		return create_save_file(session, file_total_size, file_segment_size)
	}

	return nil
}

func create_save_file(session *DownloadSession, file_total_size int, file_segment_size int) error {

	_, err := os.Create(session.save_filename)
	if err != nil {
		return err
	}

	// 创建下载文件信息
	tdg_file_name := session.save_filename + ".tdg"

	// 存在则删除
	_, err = os.Stat(tdg_file_name)
	if err == nil || os.IsExist(err) {
		if err = os.Remove(tdg_file_name); err != nil {
			return err
		}
	}

	_, err = os.Create(tdg_file_name)
	if err != nil {
		return err
	}

	// 写入下载相关信息
	tdg_file, err := os.OpenFile(tdg_file_name, os.O_RDWR, os.FileMode(0666))
	if err != nil {
		return err
	}
	defer tdg_file.Close()

	info := TDGInfo{}
	info.Version = 1
	info.TotalSize = uint32(file_total_size)
	info.FileSegmentSize = uint32(file_segment_size)

	// 各个分片的下载完成状态
	segment_count := file_total_size / file_segment_size
	if file_total_size%file_segment_size != 0 {
		segment_count++
	}

	info.SegmemtsDownloaded = make([]byte, segment_count/8+1)

	err = write_tdg_info(tdg_file, &info)
	if err != nil {
		return err
	}

	return nil
}

func read_tdg_info(session *DownloadSession) error {
	_, err := session.tdg_file.Seek(0, os.SEEK_SET)
	if err != nil {
		return errors.New("Seek failed")
	}

	err = binary.Read(session.tdg_file, binary.LittleEndian, &session.tdg_info.Version)
	if err != nil {
		return errors.New("Read Version failed")
	}

	err = binary.Read(session.tdg_file, binary.LittleEndian, &session.tdg_info.TotalSize)
	if err != nil {
		return errors.New("Read TotalSize failed")
	}

	err = binary.Read(session.tdg_file, binary.LittleEndian, &session.tdg_info.FileSegmentSize)
	if err != nil {
		return errors.New("Read FileSegmentSize failed")
	}

	segment_count := session.tdg_info.TotalSize / session.tdg_info.FileSegmentSize
	if session.tdg_info.TotalSize%session.tdg_info.FileSegmentSize != 0 {
		segment_count++
	}
	byte_count := segment_count / 8
	if segment_count%8 != 0 {
		byte_count++
	}

	session.tdg_info.SegmemtsDownloaded = make([]byte, byte_count)
	err = binary.Read(session.tdg_file, binary.LittleEndian, session.tdg_info.SegmemtsDownloaded)
	if err != nil {
		return errors.New("Read SegmemtsDownloaded failed:" + err.Error())
	}

	return nil
}

func write_tdg_info(file *os.File, info *TDGInfo) error {

	_, err := file.Seek(0, os.SEEK_SET)
	if err != nil {
		return err
	}

	// 版本号
	err = binary.Write(file, binary.LittleEndian, info.Version)
	if err != nil {
		return err
	}

	err = binary.Write(file, binary.LittleEndian, info.TotalSize)
	if err != nil {
		return err
	}

	err = binary.Write(file, binary.LittleEndian, info.FileSegmentSize)
	if err != nil {
		return err
	}

	err = binary.Write(file, binary.LittleEndian, info.SegmemtsDownloaded)
	if err != nil {
		return err
	}

	return nil
}

// 获取远程资源信息
func get_remote_res_info(urls string) HttpRespBody {

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

func getShangTickerTables(session *DownloadSession, segment_idx int) {

	defer session.wait_group.Done()

	// http 请求
	var req http.Request
	req.Method = "GET"
	req.Close = true

	var err error
	req.URL, err = url.Parse(session.remote_url)
	if err != nil {
		fmt.Println(err.Error())
		panic(err)
	}

	header := http.Header{}
	offset := segment_idx * int(session.tdg_info.FileSegmentSize)
	range_str := fmt.Sprintf("bytes=%d-%d", offset, offset+int(session.tdg_info.FileSegmentSize)-1)
	header.Set("Range", range_str)
	header.Set("User-Agent", UA)
	req.Header = header
	resp, err := http.DefaultClient.Do(&req)
	if err != nil {
		fmt.Println(err.Error())
		panic(err)
	}
	defer resp.Body.Close()

	// 互斥锁
	session.file_mutex.Lock()
	defer session.file_mutex.Unlock()

	// 实际数据长度 Content-Length:[592]
	real_length, err := strconv.Atoi(resp.Header["Content-Length"][0])
	session.save_file.Seek(int64(offset), os.SEEK_SET)
	io.CopyN(session.save_file, resp.Body, int64(real_length))

	// 更新tdg
	err = update_tdg_file(session, segment_idx)
	if err != nil {
		fmt.Println(err.Error())
	}
}

func update_tdg_file(session *DownloadSession, segment_idx int) (err error) {
	defer func() {
		if err != nil {
			err = errors.New("update_tdg_file:" + err.Error())
		}
	}()

	// 标记这部分下载完成
	// 获取分段所在字节序号（从0开始）
	which_byte := segment_idx / 8
	byte_value := session.tdg_info.SetSegmentDownloaded(segment_idx)

	// 固定头部 + which_byte
	session.tdg_file.Seek(int64(9+which_byte), os.SEEK_SET)
	err = binary.Write(session.tdg_file, binary.LittleEndian, byte_value)
	return err
}
