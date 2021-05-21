package db

import (
	mydb "filestore-server/db/mysql"
	"fmt"
	"time"
)

// UserFile : 用户文件表结构体
type UserFile struct {
	UserName    string
	FileHash    string
	FileName    string
	FileSize    int64
	UploadAt    string
	LastUpdated string
}

// OnUserFileUploadFinished : 更新用户文件表
func OnUserFileUploadFinished(username, filehash, filename string, filesize int64) bool {
	stmt, err := mydb.DBConn().Prepare("insert ignore into tbl_user_file (`user_name`,`file_sha1`,`file_name`,`file_size`,`upload_at`) values (?,?,?,?,?)")
	if err != nil {
		fmt.Println("User-File Prefare Error: ", err)
		return false
	}
	defer stmt.Close()

	_, err = stmt.Exec(username, filehash, filename, filesize, time.Now())
	if err != nil {
		fmt.Println("User-File Exec Error: ", err)
		return false
	}
	return true
}

// QueryUserFileMetas : 批量获取用户文件信息
func QueryUserFileMetas(username string, limit int) ([]UserFile, error) {
	stmt, err := mydb.DBConn().Prepare("select file_sha1,file_name,file_size,upload_at,last_update from tbl_user_file where (user_name=? and status=0) limit ?")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(username, limit)
	if err != nil {
		return nil, err
	}

	var userFiles []UserFile
	for rows.Next() {
		ufile := UserFile{}
		err = rows.Scan(&ufile.FileHash, &ufile.FileName, &ufile.FileSize, &ufile.UploadAt, &ufile.LastUpdated)
		if err != nil {
			fmt.Println(err.Error())
			break
		}
		userFiles = append(userFiles, ufile)
	}
	return userFiles, nil
}

// DeleteUserFile : 删除文件(标记删除)
func DeleteUserFile(username, filehash string) bool {
	stmt, err := mydb.DBConn().Prepare("update tbl_user_file set status=2 where user_name=? and file_sha1=? limit 1")
	if err != nil {
		fmt.Println("Delete User-File Error 1: ", err.Error())
		return false
	}
	defer stmt.Close()

	_, err = stmt.Exec(username, filehash)
	if err != nil {
		fmt.Println("Delete User-File Error 2: ", err.Error())
		return false
	}
	fmt.Println("Delete UserFile OK...")
	return true
}

// RenameFileName : 文件重命名
func RenameFileName(username, filehash, filename string) bool {
	stmt, err := mydb.DBConn().Prepare("update tbl_user_file set file_name=? where user_name=? and file_sha1=? limit 1")
	if err != nil {
		fmt.Println("RenameFile Error 1: ", err.Error())
		return false
	}
	defer stmt.Close()

	_, err = stmt.Exec(filename, username, filehash)
	if err != nil {
		fmt.Println("RenameFile Error 2: ", err.Error())
		return false
	}
	return true
}

// QueryUserFileMeta : 获取用户单个文件信息
func QueryUserFileMeta(username string, filehash string) (*UserFile, error) {
	stmt, err := mydb.DBConn().Prepare("select file_sha1,file_name,file_size,upload_at,last_update from tbl_user_file where user_name=? and file_sha1=?  limit 1")
	if err != nil {
		fmt.Println("QueryUserFileMeta Error 1: ", err)
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(username, filehash)
	if err != nil {
		fmt.Println("QueryUserFileMeta Error 2: ", err)
		return nil, err
	}

	ufile := UserFile{}
	if rows.Next() {
		err = rows.Scan(&ufile.FileHash, &ufile.FileName, &ufile.FileSize, &ufile.UploadAt, &ufile.LastUpdated)
		if err != nil {
			fmt.Println("QueryUserFileMeta Error 3: ", err)
			return nil, err
		}
	}
	fmt.Println("Query One User FileMeta OK...")
	return &ufile, nil
}
