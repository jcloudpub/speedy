# speedy usage

## chunkmaster usage

chunkmaster is a central master node designed to maintain chunkserver information and allocate the file id. 

```
$ ./chunkmaster --help
Usage of ./chunkmaster:
  -D=false: log debug level
  -d="speedy": database name
  -dh="127.0.0.1": database ip
  -dp="3306": database port
  -h="0.0.0.0": chunkmaster listen ip
  -p=8099: chunkmaster listen port
  -pw="": database passwd
  -u="root": database user

options:
-D		set -D=true log level is set to debug, default log level if info
-d		set the database used to store chunkserver info and file id, default is speedy
-dh		set the database ip, default is 127.0.0.1
-dp		set the database port, default is 3306
-h		set chunkmaster listen address, defalut is 0.0.0.0
-p		set chunkmaster listen port, default is 8099
-pw		set the password of database, default is empty
-u		set the user of database, default is root
```


## chunkserver usage

chunkserver is the final storage of all the image files, It reports self status to the chunkmaster repeatedly. normally there are multiple chunkservers to form a chunkserver group which have one unique group_id

```
$ ./spy_server
usage:spy_server
===========================
[--port=<port>]
[--ip=<listen address>]
[--data_dir=<data directory>]
[--error_log=<error log file>]
[--mem_blocks=<memory blocks for stream buffer>]
[--chunks=<number of chunks>]
[--sync=<sync when write, 1 or 0>]
[--daemonize=<1 or 0>]
--master_ip=<chunkmaster ip addr>
--master_port=<chunkmaster port>
--group_id=<unique group id>

options:
--port        set the server listen port, default is 8000
--ip          set the server listen address, default is 127.0.0.1
--data_dir    set the folder which chunk file located, default is current working directory
--error_log   set the error log file path
--sync        set whether fsync file after writing image files to chunk, default is 0
--daemonize   set whether runing server as daemon, default is 0
--master_ip   set chunkmaster listen addr so we can report chunkserver status to master node. it must be given.
--master_port set chunkmaster listen port. it must be given.
--group_id    set chunkserver unique group id which belongs to. multiple copy of chunkserver have the same unique group id

--chunks      set the amount of chunk files, default is 20. 
we use preallocated fixed size chunk file to store image data, the chunk file size is 2G right now, if --chunks=20 it means you have 40G disk space to store image data.

--mem_blocks  set the amount of uploading memory blocks, default is 1024
in order to control our server internal memory usage for uploading image, we use pre-allocated memory pool to hold the byte stream of uploading image, the memory pool consists of many fixed size memory block, the size is 1M right now, --mem_blocks actually set the number of the fixed size block. eg. if you set --mem_blocks=1024 than your uploading memory pool is 1G size.

```


## imageserver usage

imageserver is a stateless frond-end proxy server designed to provide restful api to upload and download docker image.

```
$ ./imageserver --help
Usage of ./imageserver:
  -D=false: log debug level
  -db="metadb": meta database
  -dh="127.0.0.1": metadb ip
  -dp=3306: metadb port
  -h="0.0.0.0": imageserver listen ip
  -mh="127.0.0.1": chunkmaster ip
  -mp=8099: chunkmaster port
  -n=2: the limit num of available chunkserver each chunkserver group
  -p=6788: imageserver listen port
  -pw="": metadb password
  -u="root": metadb user

options:
-D		set -D=true log level is set to debug, default log level is info
-db		set the database of meta(instead of metaserver), defalut database is metadb
-db		set the ip of meta database(metaserver ip), default is 127.0.0.1
-dp		set the port of meta database(metaserver port), default is 3306
-h		set the imageserver listen address, default is 0.0.0.0
-mh		set the chumaster address, defalut is 127.0.0.1
-mp		set the chunmaster port, default is 8099
-n		set the mininum of available chunkserver each chunkserver group, default is 2
if the num of available chunkserver < n, the chunkserver group can not be used to upload file

-p		set the imageserver listen port, default is 6788
-pw		set the password of meta database, default is empty
-u		set the user of meta database, default is root
```
