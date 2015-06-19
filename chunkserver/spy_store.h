#ifndef __SPY_STORE_H__
#define __SPY_STORE_H__

#include "spy_work.h"
#include "spy_server.h"

#define SUPERBLOCK_MAGIC "CHUNK_SB_MAGIC!"
#define SUPERBLOCK_MAGIC_SIZE 15

//SUPERBLOCK_MAGIC_SIZE + chunk_id + files_alloc + files_count + offset
//                      + avail_space + checksum
#define SUPERBLOCK_SIZE (SUPERBLOCK_MAGIC_SIZE + 8 + 4 + 4 + 8 + 8 + 8)

#define MAX_BLOCK_SIZE 4194304
#define BUFFER_SIZE    4194304
#define FILE_META_SIZE 30 // see spy_file_t
#define CHUNK_FILENAME_LEN_MAX 32

typedef struct {
	uint64_t                        chunk_id;
	uint64_t                        fid;
	uint64_t                        offset;
	uint32_t                        size;

	struct hlist_node               f_hash;
} spy_file_index_entry_t;

//FIXME: using uint32_t for chunk size, support max 4G chunk
typedef struct {
	int                             fd;
	char                           *path;
	uint64_t                        chunk_id;
	uint32_t                        files_alloc;
	uint32_t                        files_count;
	uint64_t                        current_offset;
	uint64_t                        avail_space;
	uint64_t                        file_size;
	struct list_head                c_list;
	struct hlist_node               c_hash;
} spy_chunk_t;

typedef struct {
	uint64_t                        fid;
	uint16_t                        ref;
	uint32_t                        size;
	uint64_t                        checksum;
	uint64_t                        timestamp;	
} spy_file_t;

typedef struct {
	int                             fd;
	int                             fpos;
	uint64_t                        chunk_id;
	uint64_t                        chunk_offset;

	char                           *buf;            // read buffer
	int                             buf_size;       // buffer size
	int                             buf_pos;        // current pos
	int                             buf_left;       // buffer left data
} spy_file_parser_t;

typedef struct {
	int                 retcode;

	spy_connection_t   *conn;
	spy_work_t          work;
	spy_chunk_t        *chunk;

	char                chunk_name[CHUNK_FILENAME_LEN_MAX];
} spy_dump_job_t;

typedef struct {
	int                             retcode;

	union {
		uint64_t                    offset;
		spy_file_index_entry_t     *index_entry;
	};
	
	spy_connection_t               *conn;
	spy_file_t                      file;
	spy_chunk_t                    *chunk;

	spy_work_t                      work;

	struct list_head                pw_list;
	time_t                          req_time;

	struct list_head                oc_list;       // obj cache list
} spy_io_job_t;

void spy_create_or_recover_files(char *dir);
void spy_write_file(spy_work_t *work);
void spy_read_file(spy_work_t *work);
void spy_check_chunk(spy_work_t *work);
void spy_dump_chunkfile(spy_work_t *work);

int spy_write_chunk_superblock(spy_chunk_t *chunk);
int spy_flush_and_sync_chunk(spy_chunk_t *chunk);

#endif
