#!/bin/bash
make build
# 定义需要打包的程序目录
program_dir="output"
# 遍历程序目录下的所有文件和文件夹
for file in $(ls $program_dir)
do
    result=$(echo ${file} | grep "windows")
    if [[ "${result}" != "" ]]; then
      mv "${program_dir}/$file" driver-box.exe
      # 如果是文件，则直接打包成tar包
      zip -r "${file/\.exe/}.zip" driver-box.exe driver-config
      rm driver-box.exe
      mv "${file/\.exe/}.zip" ${program_dir}
    else
      mv "${program_dir}/$file" driver-box
      # 如果是文件，则直接打包成tar包
      tar -cvf "${file}.tar" driver-box driver-config
      rm driver-box
      mv ${file}.tar ${program_dir}
    fi
done
