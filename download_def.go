package main

import (
	"os"
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

// 保存的文件信息
type TDGInfo struct {

	// 版本号 1
	Version uint8

	// 文件总大小 4
	TotalSize uint32

	// 文件分片大小 4
	FileSegmentSize uint32

	// 各个文件分片的下载状态
	// 位图，计数从0开始
	SegmemtsDownloaded []byte
}

func (this *TDGInfo) IsSegmentDownloaded(segment_idx int) bool {
	byte_idx := segment_idx / 8
	offset := uint(segment_idx % 8)

	byte_value := this.SegmemtsDownloaded[byte_idx]
	return (byte_value & (1 << offset)) != 0
}

func (this *TDGInfo) SetSegmentDownloaded(segment_idx int) byte {
	byte_idx := segment_idx / 8
	offset := uint(segment_idx % 8)

	byte_value := this.SegmemtsDownloaded[byte_idx]
	new_byte_value := byte_value | (1 << offset)
	this.SegmemtsDownloaded[byte_idx] = new_byte_value

	return new_byte_value
}

// 下载信息
type DownloadSession struct {
	// 现在文件的url
	remote_url string
	// 本地保存文件名
	save_filename string

	// tdg 信息
	tdg_info TDGInfo

	// tdg 文件
	tdg_file *os.File

	// 实际文件
	save_file *os.File

	// 由一组协程管理下载
	wait_group sync.WaitGroup

	// 文件相关的互斥锁
	file_mutex sync.Mutex
}
