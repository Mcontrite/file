package handler

import (
	"encoding/json"
	dblayer "filestore-server/db"
	"filestore-server/meta"
	"fmt"
	"net/http"
	"os"
	"path"
	"strconv"
)

// FileQueryHandler : 查询批量的文件元信息
func FileQueryHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	limitCnt, _ := strconv.Atoi(r.Form.Get("limit"))
	username := r.Form.Get("username")
	//fileMetas, _ := meta.GetLastFileMetasDB(limitCnt)
	userFiles, err := dblayer.QueryUserFileMetas(username, limitCnt)
	if err != nil {
		fmt.Println("File Query Error 1: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(userFiles)
	if err != nil {
		fmt.Println("File Query Error 2: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	fmt.Println("File Query Handler OK...")
	w.Write(data)
}

// GetFileMetaHandler : 获取文件元信息
func GetFileMetaHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	filehash := r.Form["filehash"][0]
	//fMeta := meta.GetFileMeta(filehash)
	fMeta, err := meta.GetFileMetaDB(filehash)
	if err != nil {
		fmt.Println("File Meta Handler Error 1...")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if fMeta != nil {
		data, err := json.Marshal(fMeta)
		if err != nil {
			fmt.Println("File Meta Handler Error 2: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		fmt.Println("File Meta Handler OK...")
		w.Write(data)
	} else {
		fmt.Println("File Meta Handler Error 3...")
		w.Write([]byte(`{"code":-1,"msg":"no such file"}`))
	}
}

// FileMetaUpdateHandler ： 更新元信息接口(重命名)
func FileMetaUpdateHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	opType := r.Form.Get("op")
	fileSha1 := r.Form.Get("filehash")
	username := r.Form.Get("username")
	newFileName := r.Form.Get("filename")

	if opType != "0" || len(newFileName) < 1 {
		fmt.Println("opType != 0 || len(newFileName) < 1")
		w.WriteHeader(http.StatusForbidden)
		return
	}
	if r.Method != "POST" {
		fmt.Println("r.Method != POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// 更新用户文件表tbl_user_file中的文件名，tbl_file的文件名不用修改
	_ = dblayer.RenameFileName(username, fileSha1, newFileName)

	// 返回最新的文件信息
	userFile, err := dblayer.QueryUserFileMeta(username, fileSha1)
	if err != nil {
		fmt.Println("Update Meta QueryUserFileMeta Error: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	data, err := json.Marshal(userFile)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	fmt.Println("File Meta Update Handler OK...")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// FileDeleteHandler : 删除文件及元信息
func FileDeleteHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	username := r.Form.Get("username")
	fileSha1 := r.Form.Get("filehash")

	fm, err := meta.GetFileMetaDB(fileSha1)
	if err != nil {
		fmt.Println("Delete Handler GetFileMetaDB Error: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 删除本地文件
	fsuffix := path.Ext(fm.FileName)
	err = os.Remove("./tmp/" + fm.FileSha1 + fsuffix)
	if err != nil {
		fmt.Println("os.Remove File Error: ", err)
	}
	// TODO: 可考虑删除Ceph/OSS上的文件
	// 可以不立即删除，加个超时机制，
	// 比如该文件10天后也没有用户再次上传，那么就可以真正的删除了

	// 删除文件表中的一条记录
	suc := dblayer.DeleteUserFile(username, fm.FileSha1)
	if !suc {
		fmt.Println("Delete Handler Failed...")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	fmt.Println("Delete Handler Succeed...")
	w.WriteHeader(http.StatusOK)
}

// // FileDeleteHandler : 删除文件及元信息
// func FileDeleteHandler(w http.ResponseWriter, r *http.Request) {
// 	r.ParseForm()
// 	fileSha1 := r.Form.Get("filehash")

// 	fMeta := meta.GetFileMeta(fileSha1)
// 	// 删除文件
// 	os.Remove(fMeta.Location)
// 	// 删除文件元信息
// 	meta.RemoveFileMeta(fileSha1)
// 	// TODO: 删除表文件信息

// 	fmt.Println("Delete File OK...")
// 	w.WriteHeader(http.StatusOK)
// }

// // FileMetaUpdateHandler ： 更新元信息接口(重命名)
// func FileMetaUpdateHandler(w http.ResponseWriter, r *http.Request) {
// 	r.ParseForm()

// 	opType := r.Form.Get("op")
// 	fileSha1 := r.Form.Get("filehash")
// 	newFileName := r.Form.Get("filename")

// 	if opType != "0" {
// 		w.WriteHeader(http.StatusForbidden)
// 		return
// 	}
// 	if r.Method != "POST" {
// 		w.WriteHeader(http.StatusMethodNotAllowed)
// 		return
// 	}

// 	curFileMeta := meta.GetFileMeta(fileSha1)
// 	curFileMeta.FileName = newFileName
// 	meta.UpdateFileMeta(curFileMeta)

// 	// TODO: 更新文件表中的元信息记录

// 	data, err := json.Marshal(curFileMeta)
// 	if err != nil {
// 		w.WriteHeader(http.StatusInternalServerError)
// 		return
// 	}
// 	fmt.Println("File Meta Update Handler OK...")
// 	w.WriteHeader(http.StatusOK)
// 	w.Write(data)
// }
