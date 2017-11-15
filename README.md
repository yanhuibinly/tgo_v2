# tgo_v2

使用多个package

重点加入了zipkin

log
  待开发

config
  拆分
  
  支持开关
  
  支持当前文件夹下的configs，和上一级目录
  
dao
  mysql 使用driver自带pool,支持多库

  mongo 支持多库

  加入了http

error
  
  自定义terror
  
  各层直接传递错误使用terror
  
util

  放一些工具类的
