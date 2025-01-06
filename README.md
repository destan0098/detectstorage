# Check Mass storage and send to zabbix server
![Static Badge](https://img.shields.io/badge/Go-100%25-brightgreen)
## Description

This Tool Check Mass storage and send to zabbix server




## Table of Contents



- [Usage](#usage)




## Usage

To Run make build file 
```
go build detectstorage.go
```
and add to zabbix_agentd.conf file like 
```
for linux
UserParameter=usb.status,~/GolandProjects/memorydetect/usb_monitor
for widows
UserParameter=usb.status,c:\zabbix\usb_monitor.exe

```
add item to host in zabbix server with usb.status tag





---



## Features

Check Mass storage and send to zabbix server
