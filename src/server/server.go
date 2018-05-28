package server

import (
	"net/http"
	"log"
	"fmt"
	"os"
	"io"
	"imagesStorage/src/utils"
	"strings"
	"github.com/satori/go.uuid"
	"imagesStorage/src/config"
	"time"
	"imagesStorage/src/data"
	"encoding/json"
	"github.com/syndtr/goleveldb/leveldb"
	"io/ioutil"
)

func StartServer(address string, port string) error {
	http.HandleFunc("/", index)
	http.HandleFunc("/upload", uploadHandle)
	http.HandleFunc("/delete", deleteHandle)
	return http.ListenAndServe(address + ":" + port, nil)
}

func index(w http.ResponseWriter, r *http.Request) {
	if r.RequestURI == "/" {
		fmt.Fprintf(w, "{\"code\": 200, \"msg\": \"Service running...\"}")
	} else if strings.HasPrefix(r.RequestURI, "/images/") {
		imagesHandle(w, r)
	}
}

func uploadHandle(w http.ResponseWriter, r *http.Request) {
	if !strings.EqualFold(r.Method, "post") {
		fmt.Fprintf(w, "{\"code\": 200, \"error\": \"Error Method.\"}")
		return
	}
	// 根据字段名获取表单文件
	formFile, header, err := r.FormFile("file")

	if !utils.VerifyFileType(header.Filename) {
		fmt.Fprintf(w, "{\"code\": 200, \"error\": \"Invalid file type.\"}")
		return
	}

	if err != nil {
		log.Printf("Get form file failed: %s\n", err)
		fmt.Fprintf(w, "{\"code\": 200, \"error\": \"Get form file failed.\"}")
		return
	}
	defer formFile.Close()

	// 文件保存dir
	fileDir := fmt.Sprintf("%d", time.Now().Year()) +
		"/" +
		fmt.Sprintf("%d", int(time.Now().Month())) +
		"/" +
		fmt.Sprintf("%d", int(time.Now().Day())) +
		"/"

	if err := utils.CheckoutDir(config.GetStorageDir() + fileDir); err != nil {
		fmt.Fprintf(w, "{\"code\": 200, \"error\": \"Server error.\"}")
		return
	}
	filePath := config.GetStorageDir() + fileDir + header.Filename // 文件保存path
	gotFile, err := os.Create(filePath)
	if err != nil {
		log.Printf("Create failed: %s\n", err)
		fmt.Fprintf(w, "{\"code\": 200, \"error\": \"Create file failed.\"}")
		return
	}
	defer gotFile.Close()

	// 读取表单文件，写入保存文件
	_, err = io.Copy(gotFile, formFile)
	if err != nil {
		log.Printf("Write file failed: %s\n", err)
		fmt.Fprintf(w, "{\"code\": 200, \"error\": \"Write file failed.\"}")
		return
	}

	itemUUID := uuid.Must(uuid.NewV4()).String()
	// 记录上传数据
	item := data.UploadItem{
		UUID: itemUUID,
		FileName: header.Filename,
		Directory: fileDir,
	}
	mData, err := json.Marshal(item)
	if err != nil{
		log.Fatalln(err)
	}

	if db, err := leveldb.OpenFile(config.GetDataBase(), nil); err != nil {
		log.Println("Open Database faild.")
	} else {
		err = db.Put([]byte(itemUUID), mData, nil)
		defer db.Close()
	}

	url := config.GetURL() + "images/" + fileDir + header.Filename
	url = strings.Replace(url, "//","/", -1)
	fmt.Fprintf(w, "{\"code\": 200, \"msg\": \"Upload finished.\", \"data\":%s, \"url\":\"%s\"}", string(mData), url)
}

func deleteHandle(w http.ResponseWriter, r *http.Request)  {
	if !strings.EqualFold(r.Method, "delete") {
		fmt.Fprintf(w, "{\"code\": 200, \"error\": \"Error Method.\"}")
		return
	}
	// 删除处理
	if db, err := leveldb.OpenFile(config.GetDataBase(), nil); err != nil {
		log.Println("Open Database faild.")
		log.Println(err)
	} else {
		if bodyContent, err := ioutil.ReadAll(r.Body); err == nil {
			var f interface{}
			json.Unmarshal(bodyContent, &f)
			bodyMap := f.(map[string]interface{})
			if key, ok := bodyMap["key"].(string); ok {
				// 获取当前key对应数据
				value, err := db.Get([]byte(key), nil)
				if err == nil {
					log.Println(string(value))
					// 删除对应文件
					var f interface{}
					json.Unmarshal(value, &f)
					valueMap := f.(map[string]interface{})
					fileDir := valueMap["Directory"].(string)
					fileName := valueMap["FileName"].(string)
					if err := os.Remove(fileDir + fileName); err != nil {
						log.Println("Remove file faild, file:", fileDir + fileName)
					}
				} else {
					log.Println("Not found value by key")
					fmt.Fprintf(w, "{\"code\": 200, \"error\": \"Not found value by key.\"}")
					defer db.Close()
					return
				}

				// 删除数据
				log.Println("delte key:", key)
				err = db.Delete([]byte(key), nil)
				if err != nil {
					log.Println(err)
					fmt.Fprintf(w, "{\"code\": 200, \"error\": \"Delete value faild in database.\"}")
					defer db.Close()
					return
				}
				fmt.Fprintf(w, "{\"code\": 200, \"error\": \"Delete finished.\"}")
				defer db.Close()
			}
		}
	}
}

func imagesHandle(w http.ResponseWriter, r *http.Request)  {
	if !strings.EqualFold(r.Method, "get") {
		fmt.Fprintf(w, "{\"code\": 200, \"error\": \"Error Method.\"}")
		return
	}
	// 文件
	imagePath := r.RequestURI
	imagePath = strings.Replace(imagePath, "/images", "", 1)
	targetFile := config.GetStorageDir() + imagePath
	targetFile = strings.Replace(targetFile, "//", "/", 1)
	log.Println(targetFile)
	if utils.CheckoutIfFileExists(targetFile) {
		log.Println("file found..")
		if fileStream, err := ioutil.ReadFile(targetFile); err == nil {
			w.Write(fileStream)
		}
	} else {
		w.Header().Set("status", "404")
		fmt.Fprintf(w, "{\"code\": 404, \"error\": \"File not found.\"}")
	}
}
