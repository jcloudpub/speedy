# speedy usage

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