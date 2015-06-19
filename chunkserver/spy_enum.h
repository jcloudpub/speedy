#ifndef __SPY_ENUM_H__
#define __SPY_ENUM_H__

typedef enum {
	SUCCESS                = 0,
	SERVER_ID_MISMATCH,
	CHUNK_UNAVAILABLE,
	FILE_NOT_FOUND,
	FILE_EXISTS,
	WRITE_ERROR,
	READ_ERROR,
	INTERNAL_ERROR,
	IO_JOB_UNAVAILABLE,
	REACH_MEMORY_LIMIT,
	SERVER_READ_ONLY,
	KILLED,
	CHUNK_CHECK_ERROR            // disk error
} spy_error_t;

typedef enum {
	OPCODE_WRITE           = 0,
	OPCODE_READ,
	OPCODE_DELETE,

	OPCODE_PING            = 10, // ping pong, for heartbeat
	OPCODE_CHECK_DISK,           // disk health check

	OPCODE_SET_STATUS      = 20,
	OPCODE_GET_STATUS,

	OPCODE_KILL_PD_WR      = 30, // kill pending writes
	OPCODE_QUERY_IO_STATUS,      // query current reading count, writing count, pending writes
	OPCODE_QUERY_DETAIL_INFOS,   // query detail infos

	OPCODE_DUMP_CHUNK      = 40
} spy_opcode_t;

typedef enum {
	RECEIVE_HDR            = 0,
	RECEIVE_BODY,
	PROCESS_REQUEST,
	SEND_RSP
} spy_conn_state_t;

typedef enum {
	STATUS_RW             = 0,
	STATUS_RO,
	STATUS_PRE_RO
} spy_server_status_t;

#endif
