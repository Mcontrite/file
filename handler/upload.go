package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"time"

	cmn "filestore-server/common"
	"filestore-server/config"
	dblayer "filestore-server/db"
	"filestore-server/meta"
	"filestore-server/mq"
	"filestore-server/store/ceph"
	"filestore-server/store/oss"
	"filestore-server/util"
)

func init() {
	// 目录已存在
	if _, err := os.Stat(config.TempLocalRootDir); err == nil {
		return
	}

	// 尝试创建目录
	err := os.MkdirAll(config.TempLocalRootDir, 0744)
	if err != nil {
		fmt.Println("无法创建临时存储目录，程序将退出.Error: ", err)
		os.Exit(1)
	}
}

// UploadHandler ： 处理文件上传
func UploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// 返回上传html页面
		data, err := ioutil.ReadFile("./static/view/index.html")
		if err != nil {
			fmt.Println("Get Upload Page Error: ", err)
			io.WriteString(w, "internel server error")
			return
		}
		io.WriteString(w, string(data))
	} else if r.Method == "POST" {
		// 获取请求参数
		username := r.Form.Get("username")

		// 接收文件流及存储到本地目录
		file, head, err := r.FormFile("file")
		if err != nil {
			fmt.Printf("Upload r.FormFile() Error: %s\n", err.Error())
			return
		}
		defer file.Close()
		fmt.Println("file: ", file)
		fmt.Println("head: ", head.Header)

		// 2. 把文件内容转为[]byte
		buf := bytes.NewBuffer(nil)
		writen, err := io.Copy(buf, file)
		if err != nil {
			fmt.Printf("Upload io.Copy() Error: %s\n", err.Error())
			return
		}

		basefname := path.Base(head.Filename)
		fmt.Println("head.Filename: ", head.Filename, " basefname: ", basefname)
		fsuffix := path.Ext(head.Filename)
		fname := ""
		if len(basefname) > 30 {
			fname = basefname[len(basefname)-30:]
		} else {
			fname = basefname
		}
		// 3. 构建文件元信息
		fileMeta := meta.FileMeta{
			FileName: fname,
			FileSha1: util.Sha1(buf.Bytes()), //　计算文件sha1
			FileSize: writen,
			UploadAt: time.Now().Format("2006-01-02 15:04:05"),
		}

		// 根据文件大小判断是否分块上传
		token := r.Form.Get("token")
		if writen > config.DefaultChunkSize {
			execMutilUpload(username, token, head.Filename, fileMeta.FileSha1, fileMeta.FileSize, file)
			fmt.Println("Finished exec Muti Upload...")
			http.Redirect(w, r, "/static/view/home.html", http.StatusFound)
			return
		}

		// 4. 将文件写入临时存储位置
		fileMeta.Location = config.TempLocalRootDir + fileMeta.FileSha1 + fsuffix // 临时存储地址
		newFile, err := os.Create(fileMeta.Location)
		if err != nil {
			fmt.Printf("Upload os.Create() Error: %s\n", err.Error())
			return
		}
		defer newFile.Close()

		nByte, err := newFile.Write(buf.Bytes())
		if int64(nByte) != fileMeta.FileSize {
			fmt.Printf("Upload Save File Error 1, writtenSize:%d, fileMetaSize:%d\n", nByte, fileMeta.FileSize)
			return
		} else if err != nil {
			fmt.Printf("Upload Save File Error 2, err:%s\n", err.Error())
			return
		}

		// 游标重新回到文件头部
		newFile.Seek(0, 0)

		if config.CurrentStoreType == cmn.StoreCeph {
			// 文件写入Ceph存储
			data, _ := ioutil.ReadAll(newFile)
			cephPath := "/ceph/" + fileMeta.FileSha1
			_ = ceph.PutObject("userfile", cephPath, data)
			fileMeta.Location = cephPath
		} else if config.CurrentStoreType == cmn.StoreOSS {
			// 文件写入OSS存储
			ossPath := "oss/" + fileMeta.FileSha1
			// 判断写入OSS为同步还是异步
			if !config.AsyncTransferEnable {
				fmt.Println("FileMeta.Location 1: ", fileMeta.Location)
				err = oss.Bucket().PutObject(ossPath, newFile)
				if err != nil {
					fmt.Println("Upload OSS PutObject Error: ", err.Error())
					w.Write([]byte("Upload failed!"))
					return
				}
				fileMeta.Location = ossPath
				fmt.Println("FileMeta.Location 2: ", fileMeta.Location)
				fmt.Println("Sync OSS Upload OK...")
			} else {
				// 写入异步转移任务队列
				data := mq.TransferData{
					FileHash:      fileMeta.FileSha1,
					CurLocation:   fileMeta.Location,
					DestLocation:  ossPath,
					DestStoreType: cmn.StoreOSS,
				}
				fmt.Println("MQ File Cur Location: ", data.CurLocation, "MQ File Dest Location: ", data.DestLocation)
				pubData, err := json.Marshal(data)
				if err != nil {
					fmt.Println("Async Json Marshal Error: ", err)
				}
				pubSuc := mq.Publish(config.TransExchangeName, config.TransOSSRoutingKey, pubData)
				if !pubSuc {
					// TODO: 当前发送转移信息失败，稍后重试
					fmt.Println("RabbitMQ Publish Message Failed.")
				}
				fmt.Println("Async OSS Upload OK...")
			}
		}

		// meta.UpdateFileMeta(fileMeta)
		_ = meta.UpdateFileMetaDB(fileMeta)

		// 更新用户文件表记录
		r.ParseForm()

		suc := dblayer.OnUserFileUploadFinished(username, fileMeta.FileSha1, fileMeta.FileName, fileMeta.FileSize)
		if suc {
			fmt.Println("Save User-File Table OK...")
			http.Redirect(w, r, "/static/view/home.html", http.StatusFound)
		} else {
			fmt.Println("Save User-File Table Error...")
			w.Write([]byte("Upload Failed."))
		}
	}
}

// UploadSucHandler : 上传已完成
func UploadSucHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Upload File Finished...")
	io.WriteString(w, "Upload finished!")
}
