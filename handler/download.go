package handler

import (
	dblayer "filestore-server/db"
	"filestore-server/meta"
	"filestore-server/store/oss"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

// DownloadURLHandler : 生成文件的下载地址
func DownloadURLHandler(w http.ResponseWriter, r *http.Request) {
	filehash := r.Form.Get("filehash")
	// 从文件表查找记录
	row, err := dblayer.GetFileMeta(filehash)
	if err != nil {
		fmt.Println("Download Get Meta Error: ", err)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")             //允许访问所有域
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type") //header的类型

	fmt.Println("Download row.FileAddr.String: ", row.FileAddr.String)

	// TODO: 判断文件存在OSS，还是Ceph，还是在本地
	if strings.Contains(row.FileAddr.String, "/tmp") {
		username := r.Form.Get("username")
		token := r.Form.Get("token")
		tmpUrl := fmt.Sprintf("http://%s/file/download?filehash=%s&username=%s&token=%s", r.Host, filehash, username, token)
		fmt.Println("Create Download URL From Local tmp/ OK...")
		w.Write([]byte(tmpUrl))
	} else if strings.Contains(row.FileAddr.String, "/ceph") {
		// TODO: ceph下载url
	} else if strings.Contains(row.FileAddr.String, "oss/") {
		// oss下载url
		signedURL := oss.DownloadURL(row.FileAddr.String)
		fmt.Println("Create Download URL From OSS OK...")
		w.Write([]byte(signedURL))
	}
}

// DownloadHandler : 文件下载接口
func DownloadHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	username := r.Form.Get("username")
	fsha1 := r.Form.Get("filehash")

	fm, err := meta.GetFileMetaDB(fsha1)
	if err != nil {
		fmt.Println("Download GetFileMetaDB Error: ", err)
		return
	}
	userFile, err := dblayer.QueryUserFileMeta(username, fsha1)
	if err != nil {
		fmt.Println("Download QueryUserFileMeta Error: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")             //允许访问所有域
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type") //header的类型

	f, err := os.Open(fm.Location)
	if err != nil {
		fmt.Println("Open Download File Location Error: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		fmt.Println("Download ioutil.ReadAll Error: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// attachment表示文件将会提示下载到本地，而不是直接在浏览器中打开
	w.Header().Set("Content-Type", "application/octect-stream")
	w.Header().Set("content-disposition", "attachment; filename=\""+userFile.FileName+"\"")

	fmt.Println("Download File  OK...")
	w.Write(data)
}
