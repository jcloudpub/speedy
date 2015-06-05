## Compile

`git clone https://github.com/jcloudpub/speedy.git src/github.com/jcloudpub/speedy`

`cd src/github.com/jcloudpub/speedy`

`./boostrap.sh`

`. ./dev.env`

`make`

## Install

### install metaserver(mysql)
1. install mysql
2. create metadata table in mysql, include two datbases: speedy and metadb

     `mysql -h<ip> -P<port> -p<password> -u<user> < docs/speedy.sql`

### install chunkmaster
start chunkmaster process

     `./bin/chunkmaster`

### install chunkserver

submit chunkserver info to chunkmaster `vim serverlist.json` :

``` json
[
     {"GroupId":1,"Ip":"127.0.0.1","Port":7654},   
     {"GroupId":1,"Ip":"127.0.0.1","Port":7655},
     {"GroupId":1,"Ip":"127.0.0.1","Port":7656}
]
```

call init api of chunkmaster by curl      

     `curl -i -X POST --data @serverlist.json "http://127.0.0.1:8099/v1/chunkserver/batchinitserver"`   
     
if you want to add two groups, the example of two groups's json:
     
```json
[     
     {"GroupId":1,"Ip":"127.0.0.1","Port":7654},      
     {"GroupId":1,"Ip":"127.0.0.1","Port":7655},      
     {"GroupId":1,"Ip":"127.0.0.1","Port":7656},     
     {"GroupId":2,"Ip":"127.0.0.1","Port":7664},     
     {"GroupId":2,"Ip":"127.0.0.1","Port":7665},      
     {"GroupId":2,"Ip":"127.0.0.1","Port":7666}      
] 
```

start chunkserver process group according above setting

     `./bin/spy_server --ip=127.0.0.1 --port=7654 --data_dir=~/spy_data --error_log=./err.log --group_id=1 --master_port=<chunkmaster listen port> --master_ip=<chunkmaster listen addr>`

     `./bin/spy_server --ip=127.0.0.1 --port=7655 --data_dir=~/spy_data --error_log=./err.log --group_id=1 --master_port=<chunkmaster listen port> --master_ip=<chunkmaster listen addr>`

     `./bin/spy_server --ip=127.0.0.1 --port=7656 --data_dir=~/spy_data --error_log=./err.log --group_id=1 --master_port=<chunkmaster listen port> --master_ip=<chunkmaster listen addr>`

### install imageserver
start imageserver process

     ./imageserver

how to test speedy is ok:

     dd if=/dev/zero of=test bs=512 count=65536
     ./speedytool

how to get chunkservers' info:

     `curl "http://chunkmasterIp:chunkmasterPort/v1/chunkserver/{groupId}/groupinfo" | python -mjson.tool`

example:

     `curl  "http://127.0.0.1:8099/v1/chunkserver/1/groupinfo" | python -mjson.tool`


### install docker_registry & speedy_docker_registry_driver

1. install docker_registry dependent package

     need to install python-pip, python-devel and liblzma

2. install docker_registry

     `mkdir ~/registry`

     `tar -xzvf ./docker_registry/docker-registry-core-2.0.3.tar.gz ~/registry`
     
     `cd ~/registry/docker-registry-core-2.0.3`
     
     `python setup.py install`

     `tar -xzvf ./docker_registry/docker-registry-0.9.0.tar.gz ~/registry`

     `cd ~/registry/docker-registry-0.9.0`    
     
     `python setup.py install`

3. install speedy_docker_registry_driver

     `cd docker_registry_speedy_driver`
     `python setup.py install`

4. modify registry setting

     `cp config_sample.yml  config.yml`
     
     `vim dev.env`     
     
     `export GUNICORN_WORKERS=16`     
     
     `export SETTINGS_FLAVOR=speedy`     
     
     `export SPEEDY_TMPDIR=~/registry/temp //speedy use this dir to storage temp file`    
     
     `export DOCKER_REGISTRY_CONFIG=~/registry/docker_registry_speedy_driver/config.yml`   

     `. ./dev.env`

5. start docker registry

     `docker-registry`


