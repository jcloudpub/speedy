Introduction
============

Speedy is a backend storage of docker-registry (Docker Registry API v1).

Speedy consist of 8 main parts: docker-registry-speedy-driver, ImageServer, 
ChunkMaster, ChunkServer, ChunkServer Agent, MetaServer, Monitor and Transfer.

+ docker-registry-speedy-driver
Adapter of docker-registry and Speedy.

+ ImageServer
Provide file upload and download service. ImageServer is stateless server, 
you can start multiple ImageServer instance at the same time, each one is 
equal. ImageServer will get all ChunkServers' informations from ChunkMaster 
periodically.

+ ChunkMaster
The central control node. Hold all ChunkServers informations. Internal file
ID allocator.

+ ChunkServer
Where the data actually store. Normally, 3 ChunkServers make a group, each 
ChunkServer in the same group is equal. Speedy is a strong consistency system, 
write will route to all ChunkServers of a group at the same time, write success
mean that all the ChunkServers write success, otherwise write failed.

+ ChunkServer Agent
Agent report status of ChunkServer to ChunkMaster, like disk free space, IO status etc.

+ MetaServer
Manage file meta datas, include file meta information and file directory structure.

+ Monitor
Monitoring the health of all ChunkServers and report it to ChunkMaster.

+ Transfer
Transfer data from ChunkServer to ChunkServer. When a ChunkServer of a group failed(system 
crash, disk damaged and so on), we should add a new empty ChunkServer to this group, and 
then transfer data from a health ChunkServer of this group to it.


Quick Install
=============

+ docker-registry-speedy-driver   
cd docker-registry-speedy-driver   
python setup.py install   

+ ImageServer/ChunkMaster/MonitoMaster/MonitorWorker/ChunkAgent/ChunkTool      
cd src/github.com/speedycn    
./bootstrap.sh   
. ./dev.env   
make   

+ ChunkServer   
cd chunkserver   
make   

+ MetaServer   
You can use mysql instead.   


+ Transfer   


Startup sequence
================
1.ChunkMaster   
2.ChunkServer   
3.ChunkServer-Agent   
4.MetaServer   
5.ImageServer   
6.MonitorWorker   
7.MonitorMaster   
8.docker-registry   

After that you can push and pull docker images.


Performance Test
================

We made a performance test about upload and download of Speedy. We use 4 normal servers   
(CPU: 24 core 2G HZ; Memory: 16G; Disk: 300G 10k SAS; Ethernet: 1 Gigabit) to construct   
our test environment. 

                           imageserver [node1]
                        /          |           \
                       /           |            \
        chunk-server[node2] chunk-server[node3] chunk-server [node4]

We simply use mysql instead of our internal MetaServer, at the same time,    
chunkserver is on the default buffer io model.

+ Performance Results

<table>
<tr><td> concurrent &nbsp;</td><td> fileSize(M)&nbsp; </td><td> file count&nbsp; </td><td> upload time(seconds)&nbsp; </td><td> download time(seconds)&nbsp; </td><td> upload speed(M/s)&nbsp; </td><td> download speed(M/s)&nbsp; </td></tr>
<tr><td> 10         </td><td> 16          </td><td> 100        </td><td> 42.85                </td><td> 15.23                  </td><td> 37.34        </td><td> 105.06 </td></tr>
<tr><td> 50         </td><td> 16          </td><td> 100        </td><td> 42.82                </td><td> 16.29                  </td><td> 37.37        </td><td> 98.22 </td></tr>
<tr><td> 100        </td><td> 16          </td><td> 100        </td><td> 45.69                </td><td> 14.50                  </td><td> 35.02        </td><td> 110.34 </td></tr>
<tr><td> 10         </td><td> 16          </td><td> 500        </td><td> 214.14               </td><td> 72.45                  </td><td> 37.36        </td><td> 110.42 </td></tr>
<tr><td> 50         </td><td> 16          </td><td> 500        </td><td> 213.90               </td><td> 71.40                  </td><td> 37.40        </td><td> 112.04 </td></tr>
<tr><td> 100        </td><td> 16          </td><td> 500        </td><td> 213.92               </td><td> 71.51                  </td><td> 37.40        </td><td> 111.87 </td></tr>
<tr><td> 10         </td><td> 16          </td><td> 1000       </td><td> 427.97               </td><td> 147.78                 </td><td> 37.39        </td><td> 108.27 </td></tr>
<tr><td> 50         </td><td> 16          </td><td> 1000       </td><td> 427.79               </td><td> 146.62                 </td><td> 37.40        </td><td> 109.13 </td></tr>
<tr><td> 100        </td><td> 16          </td><td> 1000       </td><td> 427.80               </td><td> 142.81                 </td><td> 37.40        </td><td> 109.13 </td></tr>
</table>


We can easily got that download speed reach the limit of Ethernet, about 110 M/s.    
Although upload speed looks like just 1/3 of download speed, acctually upload also    
reach the Ethernet limit, that is the result of upload will concurrently write three   
chunkservers.


