#tgo_v2

使用多个package

重点加入了zipkin

log

	使用logrus+lumberjack

config

	拆分

	支持开关

	支持当前文件夹下的configs，和上一级目录

	code支持code_public和code_private两个文件

dao

 	mysql 使用driver自带pool,支持多库

  	mongo 支持多库

  	加入了http

error

  	自定义terror

  	各层直接传递错误使用terror

util

  	放一些工具类的

被墙的package地址

    google.golang.org/grpc
    下载地址：github.com/grpc/grpc-go

    google.golang.org/genproto
    下载地址：github.com/google/go-genproto

    golang.org/x/net
    下载地址：github.com/golang/net