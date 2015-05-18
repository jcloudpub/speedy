Introduction
============

Speedy is a distributed storage system designed to provide high availability, high performance, strong consistency and scalability special for docker registry(Docker Registry API v1).

Speedy has 6 key parts:

+ docker-registry-speedy-driver
This is a docker-registry backend driver for speedy, 
docker-registry-speedy-driver divides docker image layer into fixed-size chunk, each chunk is identified by an immutable global unique groupid and an immutable global unique file id.
driver will retry for the chunk instead of the image layer while reading or writing failed

+ imageserver
This is a stateless server designed to provide restful api to upload and download docker image. 
imageserver get chunkserver information and file id from chunkmaster periodically. 
imageserver choose a suitable chunkserver group to storage docker image according to chunkserver information independently, 
we can start many imageserver to provice service at the same time and docker-registry-speedy-driver can use anyone of them equally.


+ chunkmaster
This is a master of speedy designed to maintain chunkserver information and allocate the file id. 
chunkmaster store chunkserver information to mysql and keep in memory as cache, while imageserver try to get chunkserver information chunkmaster send the information in memory to imageserver.
while imageserver try to get file id, chunkmaster allocate a continuous range of file id and send to imageserver.

+ chunkserver
chunkserver stores chunks on local disk as linux files and read or write chunk data specified by groupid and fileid which send from imageserver, 
chunkserver also maintains the map of groupid and fileid to the offset of chunk.
normally, a chunkserver group is consist of 3 chunkservers,
imageserver writes data to a chunkserver group suceess means storing data to each chunkserver of the group success. 

+ metaserver
metaserver is designed to mantain map information of each chunk and docker image layer and keep the relationship of docker image and different tags.

+ transfer
transfer is designed to cope data from a chunkserver to another chunkserver.
when a chunkserver of a group failed(system crash, disk damaged and so on), we should add a new empty chunkserver to this group and transfer the data from a health chunkserver to this new chunkserver.


Quick Install
=============

+ docker-registry-speedy-driver   
cd docker-registry-speedy-driver   
python setup.py install   

+ imageserver/chunkmaster/chunktool      
cd src/github.com/speedycn    
./bootstrap.sh   
. ./dev.env   
make   

+ chunkserver   
cd chunkserver   
make   

+ metaserver   
we can use mysql instead.   


+ transfer   


Startup sequence
================
1.chunkmaster   
2.chunkserver   
3.metaserver   
4.imageserver   
5.docker-registry   

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


