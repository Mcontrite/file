package main

import (
	"bufio"
	"encoding/json"
	"filestore-server/config"
	dblayer "filestore-server/db"
	"filestore-server/mq"
	"filestore-server/store/oss"
	"fmt"
	"os"
)

// ProcessTransfer : 处理文件转移
func ProcessTransfer(msg []byte) bool {
	fmt.Println("Processing Transer Messages: ", string(msg))
	// 解析 message
	pubData := mq.TransferData{}
	err := json.Unmarshal(msg, &pubData)
	if err != nil {
		fmt.Println("ProcessTransfer Unmarshal Error: ", err.Error())
		return false
	}
	// 根据临时存储文件路径创建文件句柄
	fin, err := os.Open(pubData.CurLocation)
	if err != nil {
		fmt.Println("ProcessTransfer Open Error: ", err.Error())
		return false
	}
	// 通过文件句柄将文件读取出来上传到OSS
	err = oss.Bucket().PutObject(pubData.DestLocation, bufio.NewReader(fin))
	if err != nil {
		fmt.Println("ProcessTransfer PutObject Error: ", err.Error())
		return false
	}
	// 更新文件的存储路径到文件表
	_ = dblayer.UpdateFileLocation(pubData.FileHash, pubData.DestLocation)
	return true
}

func main() {
	if !config.AsyncTransferEnable {
		fmt.Println("Aysnc transfer didn't start, need to check config...")
		return
	}
	fmt.Println("Transfer Server is running...")
	mq.StartConsume(config.TransOSSQueueName, "", ProcessTransfer)
}
