CC = gcc
DEBUG = -g -rdynamic -ggdb -O0
CFLAGS = -D_GNU_SOURCE $(DEBUG)

SRV_OBJ = spy_event.o spy_utils.o spy_store.o spy_adler32.o spy_work.o spy_log.o spy_mem_block.o spy_rw_buffer.o spy_obj_cache.o spy_agent.o

spy_server: $(SRV_OBJ) spy_server.c
	$(CC) -g -O0 -pthread -o $@ $^

spy_utils.o: spy_utils.h spy_utils.c
spy_store.o: spy_store.h spy_store.c
spy_adler32.o: spy_adler32.c
spy_work.o: spy_work.h spy_work.c
spy_event.o: spy_event.h spy_event.c
spy_log.o: spy_log.h spy_log.c
spy_mem_block.o: spy_mem_block.h spy_mem_block.c
spy_rw_buffer.o: spy_rw_buffer.h spy_rw_buffer.c
spy_obj_cache.o: spy_obj_cache.h spy_obj_cache.c
spy_agent.o: spy_agent.h spy_agent.c

clean:
	rm -rf *.o spy_server
