package handler

import (
	"bufio"
	"bytes"
	"filestore-server/config"
	"filestore-server/util"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/garyburd/redigo/redis"
	jsonit "github.com/json-iterator/go"

	rPool "filestore-server/cache/redis"
	dblayer "filestore-server/db"
)

func execMutilUpload(username, token, basename, filehash string, fsize int64, file io.Reader) {
	fsizeStr := strconv.FormatInt(fsize, 10)
	fmt.Println("Fsize: ", fsize, " FsizeStr: ", fsizeStr)

	// 1. 请求初始化分块上传接口
	resp, err := http.PostForm(
		"http://localhost:8080/file/mpupload/init",
		url.Values{
			"username": {username},
			"token":    {token},
			"filehash": {filehash},
			"filesize": {fsizeStr},
		})
	if err != nil {
		fmt.Println("Mock http.PostForm Error 1: ", err.Error())
		os.Exit(-1)
	}
	defer resp.Body.Close()

	// 读取 body 响应结果
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Mock ioutil.ReadAll Error 0: ", err.Error())
		os.Exit(-1)
	}

	// 2. 得到uploadID以及服务端指定的分块大小chunkSize
	uploadID := jsonit.Get(body, "data").Get("UploadID").ToString()
	chunkSize := jsonit.Get(body, "data").Get("ChunkSize").ToInt()
	fmt.Printf("uploadid: %s  chunksize: %d\n", uploadID, chunkSize)

	// 3. 请求分块上传接口
	tURL := "http://localhost:8080/file/mpupload/uppart?username=root&token=" + token + "&uploadid=" + uploadID
	multipartUpload(basename, tURL, chunkSize)

	// 4. 请求分块完成接口
	resp, err = http.PostForm(
		"http://localhost:8080/file/mpupload/complete",
		url.Values{
			"username": {username},
			"token":    {token},
			"filehash": {filehash},
			"filesize": {fsizeStr},
			"filename": {basename},
			"uploadid": {uploadID},
		})
	if err != nil {
		fmt.Println("Mock http.PostForm Error 2: ", err.Error())
		os.Exit(-1)
	}
	defer resp.Body.Close()

	// 读取 body 响应结果
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Mock ioutil.ReadAll Error: ", err.Error())
		os.Exit(-1)
	}
	fmt.Printf("Muti Complete Result: %s\n", string(body))
}

func multipartUpload(file, targetURL string, chunkSize int) error {
	// 根据文件名打开文件
	filePath := ""
	if strings.Contains(file, ".jpg") || strings.Contains(file, ".png") {
		filePath = "C:/Users/miaobingzhou/Pictures/Saved Pictures/" + file
	} else {
		filePath = "D:/Zips/" + file
	}

	// 打开文件
	f, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Mock Mutil os.Open Error: ", err)
		return err
	}
	defer f.Close()
	fmt.Println("Open File OK...")

	bfRd := bufio.NewReader(f)
	index := 0
	ch := make(chan int)
	buf := make([]byte, chunkSize) //每次读取chunkSize大小的内容

	for {
		n, err := bfRd.Read(buf)
		fmt.Println("bfRd.Read(buf) Error: ", err)
		if n <= 0 {
			fmt.Println("Mock Mutil bfRd.Read(buf) < 0, index: ", index)
			break
		}
		index++

		bufCopied := make([]byte, 5*1048576)
		copy(bufCopied, buf)
		fmt.Println("Copy Buf OK...", index)

		go func(b []byte, curIdx int) {
			fmt.Println("Start A Go ", curIdx, "Upload Size: ", len(b))

			resp, err := http.Post(targetURL+"&index="+strconv.Itoa(curIdx), "multipart/form-data", bytes.NewReader(b))
			if err != nil {
				fmt.Println("Mock Mutil http.Post Error: ", err)
			}

			body, er := ioutil.ReadAll(resp.Body)
			fmt.Printf("resp.Body: %+v, resp.Body.err: %+v\n", string(body), er)
			resp.Body.Close()

			fmt.Println("End Go 1 ", curIdx)
			ch <- curIdx
			fmt.Println("End Go 2 ", curIdx)
		}(bufCopied[:n], index)
		fmt.Println("ENd Moc 1...", index)
		//遇到任何错误立即返回，并忽略 EOF 错误信息
		if err != nil {
			if err == io.EOF {
				fmt.Println("Mock Mutil err==io.EOF", err)
				break
			} else {
				fmt.Println("Mock Mutil Error: ", err.Error())
			}
		}
		fmt.Println("ENd Moc 2...", index)
	}

	for idx := 0; idx < index; idx++ {
		select {
		case res := <-ch:
			fmt.Println("Mock Index ", idx, " Res: ", res)
		}
	}

	return nil
}

// MultipartUploadInfo : 初始化信息
type MultipartUploadInfo struct {
	FileHash   string
	FileSize   int
	UploadID   string
	ChunkSize  int
	ChunkCount int
}

// InitialMultipartUploadHandler : 初始化分块上传
func InitialMultipartUploadHandler(w http.ResponseWriter, r *http.Request) {
	// 1. 解析用户请求参数
	r.ParseForm()
	username := r.Form.Get("username")
	filehash := r.Form.Get("filehash")
	filesize, err := strconv.Atoi(r.Form.Get("filesize"))
	if err != nil {
		fmt.Println("Muti Init Param Invalid: ", err)
		w.Write(util.NewRespMsg(-1, "params invalid", nil).JSONBytes())
		return
	}

	// 2. 获得redis的一个连接
	rConn := rPool.RedisPool().Get()
	defer rConn.Close()

	// 3. 生成分块上传的初始化信息
	upInfo := MultipartUploadInfo{
		FileHash:   filehash,
		FileSize:   filesize,
		UploadID:   username + fmt.Sprintf("%x", time.Now().UnixNano()),
		ChunkSize:  config.DefaultChunkSize, // 5MB
		ChunkCount: int(math.Ceil(float64(filesize) / config.DefaultChunkSize)),
	}

	// 4. 将初始化信息写入到redis缓存，HSET 命令给哈希表赋值
	rConn.Do("HSET", "MP_"+upInfo.UploadID, "chunkcount", upInfo.ChunkCount)
	rConn.Do("HSET", "MP_"+upInfo.UploadID, "filehash", upInfo.FileHash)
	rConn.Do("HSET", "MP_"+upInfo.UploadID, "filesize", upInfo.FileSize)

	fmt.Println("Init Mutipart Uplaod OK...")
	// 5. 将响应初始化数据返回到客户端
	w.Write(util.NewRespMsg(0, "OK", upInfo).JSONBytes())
}

// UploadPartHandler : 上传文件分块
func UploadPartHandler(w http.ResponseWriter, r *http.Request) {
	// 1. 解析用户请求参数
	r.ParseForm()
	//	username := r.Form.Get("username")
	uploadID := r.Form.Get("uploadid")
	chunkIndex := r.Form.Get("index")

	// 2. 获得redis连接池中的一个连接
	rConn := rPool.RedisPool().Get()
	defer rConn.Close()

	// 3. 获得文件句柄，用于存储分块内容
	fpath := "/data/" + uploadID + "/" + chunkIndex
	os.MkdirAll(path.Dir(fpath), 0744)
	fd, err := os.Create(fpath)
	if err != nil {
		fmt.Println("Muti Upload os.Create Error: ", err)
		w.Write(util.NewRespMsg(-1, "Upload part failed", nil).JSONBytes())
		return
	}
	defer fd.Close()

	buf := make([]byte, 1024*1024)
	for {
		n, err := r.Body.Read(buf)
		fd.Write(buf[:n])
		if err != nil {
			fmt.Println(err)
			break
		}
	}

	// 4. 更新redis缓存状态
	rConn.Do("HSET", "MP_"+uploadID, "chkidx_"+chunkIndex, 1)

	fmt.Println("Upload Mutipart File OK...")
	// 5. 返回处理结果到客户端
	w.Write(util.NewRespMsg(0, "OK", nil).JSONBytes())
}

// CompleteUploadHandler : 通知上传合并
func CompleteUploadHandler(w http.ResponseWriter, r *http.Request) {
	// 1. 解析请求参数
	r.ParseForm()
	upid := r.Form.Get("uploadid")
	username := r.Form.Get("username")
	filehash := r.Form.Get("filehash")
	filesize := r.Form.Get("filesize")
	filename := r.Form.Get("filename")

	// 2. 获得redis连接池中的一个连接
	rConn := rPool.RedisPool().Get()
	defer rConn.Close()

	// 3. 通过uploadid查询redis判断是否所有分块上传完成
	data, err := redis.Values(rConn.Do("HGETALL", "MP_"+upid))
	if err != nil {
		fmt.Println("Muti Complete redis.Values Error: ", err)
		w.Write(util.NewRespMsg(-1, "complete upload failed", nil).JSONBytes())
		return
	}
	totalCount := 0
	chunkCount := 0
	// 查询结果中 key 和 value都存在一起
	for i := 0; i < len(data); i += 2 {
		k := string(data[i].([]byte))
		v := string(data[i+1].([]byte))
		if k == "chunkcount" {
			totalCount, _ = strconv.Atoi(v)
		} else if strings.HasPrefix(k, "chkidx_") && v == "1" {
			chunkCount++
		}
	}
	if totalCount != chunkCount {
		fmt.Println("Muti Complete totalCount != chunkCoun Error: ", err)
		w.Write(util.NewRespMsg(-2, "invalid request", nil).JSONBytes())
		return
	}

	// 4. TODO：合并分块

	// 5. 更新唯一文件表及用户文件表
	fsize, _ := strconv.Atoi(filesize)
	dblayer.OnFileUploadFinished(filehash, filename, int64(fsize), "")
	dblayer.OnUserFileUploadFinished(username, filehash, filename, int64(fsize))

	fmt.Println("Complete MutiPart File OK...")
	// 6. 响应处理结果
	w.Write(util.NewRespMsg(0, "OK", nil).JSONBytes())
}
