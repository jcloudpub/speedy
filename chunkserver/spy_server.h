#ifndef __SPY_SERVER__
#define __SPY_SERVER__
#include <stdint.h>

#include "spy_list.h"
#include "spy_event.h"
#include "spy_work.h"
#include "spy_log.h"
#include "spy_enum.h"
#include "spy_mem_block.h"
#include "spy_rw_buffer.h"
#include "spy_agent.h"

#define CHUNK_HASH_SIZE      (1 << 10)
#define FILE_HASH_SIZE       (1 << 20)

#define DEF_CHUNK_SIZE       (1UL << 31)
#define DEF_N_CHUNKS         20
#define MAX_N_CHUNKS         (DEF_N_CHUNKS * 10)
#define DEF_LOG_PATH         "./speedy.log"
#define DEF_SERVER_ADDR      "127.0.0.1"
#define DEF_DATA_DIR         "./"
#define DEF_DAEMONIZE        0
#define DEF_SERVER_PORT      8000
#define DEF_LOG_LEVEL        INFO
#define DEF_MAX_CLIENTS      10240
#define DEF_RX_BUF_SIZE      (1 << 20)
#define DEF_SND_BUF_SIZE     (1 << 20)
#define DEF_BUF_EXPAND_SIZE  512
#define DEF_MAX_IO_JOBS      5000
#define DEF_MAX_PENDING_WRITES      20
#define DEF_PENDING_WRITE_TIMEOUT   5       // 5 seconds
#define DEF_CHECK_BLOCK_SIZE (1 << 12)      // 4k
#define DEF_DETAIL_INFOS_HDR_SIZE   76

#define REQ_HDR_SIZE            (1 + 4)     //opcode(1 byte) + body_len(4 byte) 
#define REQ_INNER_HDR_SIZE      (2 + 8)     // server_id(2 bytes) + fid(8 bytes)
#define RSP_HDR_SIZE            (1 + 1 + 4) //opcode(1) + error(1 byte) + body_len(4)

#define MIN(a, b) ((a) < (b) ? (a) : (b))
#define MAX(a, b) ((a) > (b) ? (a) : (b))

typedef struct {
	int                     fd;

	spy_rw_buffer_t         request;        // request
	spy_rw_buffer_t         rsp_body;       // response body
	unsigned char           rsp_header[RSP_HDR_SIZE]; // response header
	int                     sentlen;

	uint8_t                 opcode;
	uint8_t                 error;
	uint32_t                body_len;       // request or response body len

	spy_conn_state_t        state;
} spy_connection_t;

typedef struct {
	int                     port;
	int                     master_port;
	int                     server_id;
	int                     daemonize;
	int                     n_chunks;
	int                     log_level;

	char                   *bind_addr;
	char                   *master_addr;
	char                   *data_dir;
	char                   *log_path;

	unsigned long long      chunk_size;

	size_t                  mb_prealloc_count;
	size_t                  mb_limit;

	int                     sync;
} spy_config_t;

typedef struct {
	int                     log_fd;
	int                     listen_fd;
	int                     n_chunks;
	uint64_t                max_chunk_id;

	uint32_t                mem_blocks_alloc;
	uint32_t                mem_blocks_alloc_limit;
	uint32_t                mem_blocks_used;

	aeEventLoop            *event_loop;

	struct list_head        chunks;
	struct list_head        writing_chunks;

	struct list_head        free_mem_blocks;

	struct hlist_head       chunk_index[CHUNK_HASH_SIZE];
	struct hlist_head       file_index[FILE_HASH_SIZE];

	spy_work_queue_t       *wq;

	struct list_head        pending_writes;
	uint32_t                pending_writes_size;

	struct list_head        free_io_jobs;

	spy_server_status_t     status;
} spy_server_t;

typedef struct {
	uint64_t                read_bytes;
	uint64_t                write_bytes;

	uint64_t                read_count;
	uint64_t                write_count;
	uint64_t                read_error;
	uint64_t                write_error;

	uint32_t                conn_count;
	uint32_t                reading_count;
	uint32_t                writing_count;
} spy_statistic_t;

extern spy_config_t         config;
extern spy_server_t         server;
extern spy_statistic_t      statistic;
extern spy_report_info_t    report_info;
#endif
