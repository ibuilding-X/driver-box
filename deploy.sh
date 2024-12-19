#!/bin/bash

# 定义需要打包的程序目录
program_dir="_output"
app_name="driver-box"

# 执行交叉编译
rm -rf ${program_dir}
export GOPROXY=https://mirrors.aliyun.com/goproxy/,direct
go mod tidy
go mod vendor

build(){
    GOOS=$1
    GOARCH=$2
    output_flag="${program_dir}/${app_name}-${GOOS}-${GOARCH}"
    if [ "$GOOS" = "windows" ]; then
      output_flag="${program_dir}/${app_name}-${GOOS}-${GOARCH}.exe"
    fi

    GOOS=$GOOS GOARCH=${GOARCH} go build  -o ${output_flag} main.go
    # 添加错误处理，确保在构建失败时脚本能够退出
    if [ $? -ne 0 ]; then
      echo "构建失败: ${output_flag}"
      exit 1
    fi

    echo "成功构建: ${output_flag}"
}
#make build VERSION=${VERSION} BuildTime=$(date +%Y%m%d%H%M%S)
build linux arm64
build linux amd64
build linux arm

#build windows amd64
#build windows arm64
#build darwin amd64
#build darwin arm64


# 遍历程序目录下的所有文件和文件夹
for file in $(ls $program_dir)
do
    deploy_file=''
    rm -rf driver-box
    mkdir driver-box
    cp -R res driver-box

    result=$(echo ${file} | grep "windows")
    if [[ "${result}" != "" ]]; then
      mv "${program_dir}/$file" driver-box/driver-box.exe
      # 如果是文件，则直接打包成tar包
      deploy_file="${file%.*}-${VERSION}.zip"
      zip -r ${deploy_file} driver-box
    else
      mv "${program_dir}/$file" driver-box/driver-box
      # 如果是文件，则直接打包成tar包
      deploy_file="${file}-${VERSION}.tar.gz"
      tar -czf ${deploy_file} driver-box
    fi
    mv ${deploy_file} ${program_dir}/
done
rm -rf driver-box