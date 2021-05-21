package config

import (
	cmn "filestore-server/common"
)

const (
	// 设置当前文件的存储类型
	CurrentStoreType = cmn.StoreOSS

	// TempLocalRootDir : 本地临时存储地址的路径
	TempLocalRootDir = "./tmp/"

	// UploadServiceHost : 上传服务监听的地址
	UploadServiceHost = "0.0.0.0:8080"

	DefaultChunkSize = 5 * 1024 * 1024
)
