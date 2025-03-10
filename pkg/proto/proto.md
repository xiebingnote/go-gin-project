# Protobuf 安装使用说明
  protobuf 是一种用于描述结构化数据的语言，它使用一种类似于C++的语法来描述数据结构，然后编译成C++、Java、Python、Go、Ruby、JavaScript、PHP、C#、Dart、Objective-C、Swift等语言的多个实现。
## protobuf 安装说明：
### mac：
    1、安装命令：
    brew install protobuf
    2、版本检查：
    protoc --version
    3、安装golang插件：
    go install google.golang.org/protobuf/cmd/protoc-gen-go
    4、添加 $GOPATH/bin 到 PATH：
    在 ~/.zshrc 或 ~/.bash_profile 文件中添加如下内容：
    export PATH="$PATH:$(go env GOPATH)/bin"
    5、切换到对应的proto文件目录中，执行命令：
     protoc --proto_path=. --go_out=. *.proto
    
    解决：--go_out: protoc-gen-go: Plugin failed with status code 1.
    原因：Protobuf 和 protoc-gen-go 插件的版本不兼容导致
    解决方法：
        方法1：保持版本一致，升级或降级到适配版本
        方法2：在proto文件中添加：option go_package = "/proto";（会在目录下生成proto文件夹）或者：option go_package = "./";（不会生成proto文件夹）