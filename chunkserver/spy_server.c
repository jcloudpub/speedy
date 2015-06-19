#include <sys/resource.h>
#include <assert.h>
#include <string.h>
#include <errno.h>
#include <signal.h>
#include <stdlib.h>
#include <unistd.h>
#include <stdio.h>
#include <fcntl.h>
#include <sys/socket.h>
#include <sys/eventfd.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <stddef.h>
#include <inttypes.h>
#include <getopt.h>

#include "spy_utils.h"
#include "spy_log.h"
#include "spy_list.h"
#include "spy_event.h"
#include "spy_server.h"
#include "spy_store.h"
#include "spy_mem_block.h"
#include "spy_obj_cache.h"
#include "spy_agent.h"

spy_config_t config;
spy_server_t server;
spy_statistic_t statistic;

spy_report_info_t report_info;

static const char server_options[] = "m:r:p:h:w:e:g:b:c:f:d";
const struct option server_long_options[] = {
	{"ip",          1, NULL, 'h'},
	{"master_ip",   1, NULL, 'm'},
	{"port",        1, NULL, 'p'},
	{"master_port", 1, NULL, 'r'},
	{"data_dir",    1, NULL, 'w'},
	{"error_log",   1, NULL, 'e'},
	{"group_id",    1, NULL, 'g'},
	{"mem_blocks",  1, NULL, 'b'},
	{"chunks",      1, NULL, 'c'},
	{"sync",        1, NULL, 'f'},
	{"daemonize",   1, NULL, 'd'},
	{NULL, 0, NULL, 0}
};
//forward declare
static int spy_set_blocking(int fd, int block);
static void spy_send_response_if_needed(spy_connection_t *conn);
static void spy_free_client_connection(spy_connection_t *conn);
static void spy_reply_body(spy_connection_t *conn);
static void spy_reply_header(spy_connection_t *conn);
static void spy_read_handler(aeEventLoop *el, int fd, void *priv,int mask);
static void spy_lookup_file_index(uint64_t fid, spy_file_index_entry_t **i,
                                  struct hlist_head **head);

static void spy_reset_client_connection(spy_connection_t *conn)
{
	int n;

	spy_remove_file_event(server.event_loop, conn->fd, AE_WRITABLE);

	// reset request&&rsp_body buffer and take mem blocks back to free list
	spy_rw_buffer_reset(&conn->request);
	spy_rw_buffer_reset(&conn->rsp_body);

	conn->sentlen   = 0;
	conn->error     = 0;
	conn->body_len  = 0;
	conn->state     = RECEIVE_HDR;

	n = spy_create_file_event(server.event_loop, conn->fd, AE_READABLE,
						  spy_read_handler, conn);

	assert(n != AE_ERR);
}

static void spy_write_handler(aeEventLoop *el, int fd, void *priv, int mask)
{
	int nwritten, remaining;
	char *snd_buf;
	size_t len;

	spy_connection_t *conn = priv;

	// TODO: merge send rsp_header and rsp_body

	remaining = RSP_HDR_SIZE + conn->body_len - conn->sentlen;

	while (remaining > 0) {

		if (conn->sentlen < RSP_HDR_SIZE) { 
			// send header
			snd_buf = conn->rsp_header + conn->sentlen;
			len     = RSP_HDR_SIZE - conn->sentlen;

			nwritten = write(conn->fd, snd_buf, len);

			if (nwritten < 0 && errno == EAGAIN)
				return;

			if (nwritten < 0) {
				spy_log(ERROR, "write cli err:%s", strerror(errno));

				spy_free_client_connection(conn);
				return;
			}
		} else { 
			// send body
			assert (spy_rw_buffer_next_readable(&conn->rsp_body, &snd_buf, &len) == 0);

			if (len > remaining)
				len = remaining;

			nwritten = write(conn->fd, snd_buf, len);
			
			if (nwritten < 0 && errno == EAGAIN)
				return;

			if (nwritten < 0) {
				spy_log(ERROR, "write cli err:%s", strerror(errno));

				spy_free_client_connection(conn);
				return;
			}

			conn->rsp_body.read_pos += nwritten;
		}

		conn->sentlen += nwritten;

		remaining = RSP_HDR_SIZE + conn->body_len - conn->sentlen;
	}

	if (conn->sentlen == RSP_HDR_SIZE + conn->body_len)
		spy_reset_client_connection(conn);
}

static void spy_send_response_if_needed(spy_connection_t *conn)
{
	int n;

	if (conn->state == SEND_RSP) {
		n = spy_create_file_event(server.event_loop, conn->fd,
                                  AE_WRITABLE, spy_write_handler, conn);

		if (n == AE_ERR) {
			spy_log(ERROR, "create write event failed");
			spy_free_client_connection(conn);		
		}
	}
}

static void spy_free_client_connection(spy_connection_t *conn)
{
	// take mem blocks back to free list
	spy_rw_buffer_reset(&conn->request);
	spy_rw_buffer_reset(&conn->rsp_body);

	if (conn->fd) {
		close(conn->fd);
		spy_remove_file_event(server.event_loop, conn->fd, 
                              AE_READABLE | AE_WRITABLE);
	}

	statistic.conn_count--;

	free(conn);
}

static void spy_reply_body(spy_connection_t *conn)
{
	spy_send_response_if_needed(conn);
}

static void spy_reply_header(spy_connection_t *conn)
{	
	char *header = conn->rsp_header;

	spy_mach_write_to_1(header, conn->opcode);
	header += 1;

	spy_mach_write_to_1(header, conn->error);
	header += 1;

	spy_mach_write_to_4(header, conn->body_len);
	header += 4;

	spy_send_response_if_needed(conn);
}

static void spy_read_file_done(spy_work_t *work)
{
	spy_file_t *file;
	spy_connection_t *conn;

	spy_io_job_t *io_job = container_of(work, spy_io_job_t, work);

	file = &io_job->file;
	conn = io_job->conn;

	statistic.reading_count--;

	if (io_job->retcode != 0) {
		conn->error    = READ_ERROR;
		conn->state    = SEND_RSP;
		conn->body_len = 0;
		spy_reply_header(conn);

		statistic.read_error++;

		spy_free_io_job(io_job);
		return;
	}

	conn->error    = SUCCESS;
	conn->body_len = file->size;
	spy_reply_header(conn);

	conn->state    = SEND_RSP;
	spy_reply_body(conn);	

	statistic.read_bytes += file->size;

	spy_free_io_job(io_job);
}

static void spy_write_file_done(spy_work_t *work)
{
	spy_file_t *file;
	spy_chunk_t *chunk;
	spy_connection_t *conn;
	struct hlist_head *head;
	spy_file_index_entry_t *index_entry, *same_entry = NULL;

	spy_io_job_t *tmp, *pending_io_job = NULL;
	spy_io_job_t *io_job = container_of(work, spy_io_job_t, work);

	conn           = io_job->conn;
	file           = &io_job->file;
	chunk          = io_job->chunk;
	conn->body_len = 0;

	statistic.writing_count--;

	// avoid duplicate fid
	spy_lookup_file_index(file->fid, &same_entry, &head);
	if (same_entry) {
		spy_log(ERROR, "internal error: same index entry");
		conn->error = INTERNAL_ERROR;

		goto ERROR_REPLY;
	}

	// deal pending writes
	// DEF_MAX_PENDING_WRITES is small, so foreach will run fast
	if (server.pending_writes_size > 0) {
		list_for_each_entry(tmp, &server.pending_writes, pw_list) {
			if (chunk->avail_space > tmp->file.size + FILE_META_SIZE) {
				pending_io_job = tmp;
				break;
			}
		}
	}

	if (pending_io_job) {
		list_del(&pending_io_job->pw_list);
		server.pending_writes_size--;

		pending_io_job->chunk       = chunk;
		pending_io_job->offset      = chunk->current_offset;

		statistic.write_count++;
		statistic.writing_count++;

		spy_queue_work(server.wq, &pending_io_job->work);
	} else {
		//return chunk to alloc list
		list_move_tail(&chunk->c_list, &server.chunks);
	}

	//spy_rw_buffer_reset(&conn->request);

	if (io_job->retcode != 0) {
		conn->error = WRITE_ERROR;

		goto ERROR_REPLY;
	}

	statistic.write_bytes += file->size;

	index_entry = malloc(sizeof(spy_file_index_entry_t));
	assert(index_entry);

	index_entry->chunk_id = chunk->chunk_id;
	index_entry->fid      = file->fid;
	index_entry->offset   = io_job->offset;
	index_entry->size     = file->size + FILE_META_SIZE;

	hlist_add_head(&index_entry->f_hash, head);

	conn->error = SUCCESS;
	conn->state = SEND_RSP;
	spy_reply_header(conn);	

	spy_free_io_job(io_job);

	return;

ERROR_REPLY:
	statistic.write_error++;

	conn->state = SEND_RSP;
	spy_reply_header(conn);

	spy_free_io_job(io_job);
}

static void spy_check_chunk_done(spy_work_t *work)
{
	spy_io_job_t *io_job   = container_of(work, spy_io_job_t, work);
	spy_connection_t *conn = io_job->conn;

	conn->error    = (io_job->retcode == 0) ? SUCCESS : CHUNK_CHECK_ERROR;
	conn->body_len = 0;
	conn->state    = SEND_RSP;
	spy_reply_header(conn);

	spy_free_io_job(io_job);
}

static void spy_kill_pending_writes(spy_connection_t *conn)
{
	spy_io_job_t *pending_wr;
	spy_connection_t *pending_conn;

	while (!list_empty(&server.pending_writes)) {

		pending_wr = list_first_entry(&server.pending_writes, spy_io_job_t, pw_list);

		list_del(&pending_wr->pw_list);
		server.pending_writes_size --;

		pending_conn = pending_wr->conn;

		pending_conn->error    = KILLED;
		pending_conn->body_len = 0;
		pending_conn->state    = SEND_RSP;

		spy_reply_header(pending_conn);

		spy_free_io_job(pending_wr);
	}

	conn->error    = 0;
	conn->body_len = 0;
	conn->state    = SEND_RSP;

	spy_reply_header(conn);
}

static void spy_query_io_status(spy_connection_t *conn)
{
	char body[12]; // pending_writes(4), writing_count(4), reading_count(4)
	char *ptr = body;

	while (conn->rsp_body.cap < 12) {
		if (spy_rw_buffer_expand(&conn->rsp_body) < 0) {
			spy_log(ERROR, "expand rsp_body failed. reach mem blocks limit.");

			conn->error = REACH_MEMORY_LIMIT;
			goto ERROR_REPLY;
		}
	}

	conn->error    = 0;
	conn->body_len = 12;
	spy_reply_header(conn);

	spy_mach_write_to_4(ptr, server.pending_writes_size);
	ptr += 4;

	spy_mach_write_to_4(ptr, statistic.writing_count);
	ptr += 4;

	spy_mach_write_to_4(ptr, statistic.reading_count);
	ptr += 4;

	assert (spy_rw_buffer_write_n(&conn->rsp_body, body, 12) == 12);

	conn->state = SEND_RSP;
	spy_reply_body(conn);

	return;

ERROR_REPLY:
	conn->body_len = 0;
	conn->state    = SEND_RSP;

	spy_reply_header(conn);
}

static void spy_query_detail_infos(spy_connection_t *conn)
{
	// conn_count(4), reading_count(4), writing_count(4), pending_write(4)
	// read_count(8), write_count(8), read_error(8), write_error(8), 
	// read_bytes(8), write_bytes(8),
	// n_chunks(4), [(chunkid(8), chunk_avail(8)), ...]
	// total_file_count(8)
	spy_chunk_t *tmp;
	size_t size;
	char *buf = NULL, *ptr;
	uint64_t files_count = 0;

	size = DEF_DETAIL_INFOS_HDR_SIZE 
			+ server.n_chunks * (sizeof(tmp->chunk_id) + sizeof(tmp->avail_space));

	buf = (char*)malloc(size);
	assert (buf);

	ptr = buf;

	spy_mach_write_to_4(ptr, statistic.conn_count);
	ptr += 4;

	spy_mach_write_to_4(ptr, statistic.reading_count);
	ptr += 4;

	spy_mach_write_to_4(ptr, statistic.writing_count);
	ptr += 4;

	spy_mach_write_to_4(ptr, server.pending_writes_size);
	ptr += 4;

	spy_mach_write_to_8(ptr, statistic.read_count);
	ptr += 8;

	spy_mach_write_to_8(ptr, statistic.write_count);
	ptr += 8;

	spy_mach_write_to_8(ptr, statistic.read_error);
	ptr += 8;

	spy_mach_write_to_8(ptr, statistic.write_error);
	ptr += 8;

	spy_mach_write_to_8(ptr, statistic.read_bytes);
	ptr += 8;

	spy_mach_write_to_8(ptr, statistic.write_bytes);
	ptr += 8;

	spy_mach_write_to_4(ptr, (uint32_t)server.n_chunks);
	ptr += 4;

	// list all chunks avail
	list_for_each_entry(tmp, &server.chunks, c_list) {
		spy_mach_write_to_8(ptr, tmp->chunk_id);
		ptr += 8;

		spy_mach_write_to_8(ptr, tmp->avail_space);
		ptr += 8;

		files_count += tmp->files_count;
	}

	list_for_each_entry(tmp, &server.writing_chunks, c_list) {
		spy_mach_write_to_8(ptr, tmp->chunk_id);
		ptr += 8;

		spy_mach_write_to_8(ptr, tmp->avail_space);
		ptr += 8;

		files_count += tmp->files_count;
	}

	spy_mach_write_to_8(ptr, files_count);
	ptr += 8;

	while (conn->rsp_body.cap < size) {
		if (spy_rw_buffer_expand(&conn->rsp_body) < 0) {
			spy_log(ERROR, "expand rsp_body failed. reach mem blocks limit.");

			conn->error = REACH_MEMORY_LIMIT;
			goto ERROR_REPLY;
		}
	}

	conn->error    = 0;
	conn->body_len = size;
	spy_reply_header(conn);

	assert (spy_rw_buffer_write_n(&conn->rsp_body, buf, size) == size);

	conn->state = SEND_RSP;
	spy_reply_body(conn);

	if (buf)
		free(buf);

	return;

ERROR_REPLY:
	conn->body_len = 0;
	conn->state    = SEND_RSP;

	spy_reply_header(conn);

	if (buf)
		free(buf);
}

static void spy_handle_ping(spy_connection_t *conn)
{
	spy_remove_file_event(server.event_loop, conn->fd, AE_READABLE);

	conn->body_len = 0;
	conn->error = SUCCESS;
	conn->state = SEND_RSP;

	spy_reply_header(conn);
}

static void spy_handle_write(spy_connection_t *conn)
{
	uint16_t server_id;
	uint64_t fid;
	uint32_t body_len;
	size_t n;
	char header[REQ_INNER_HDR_SIZE];
	char *buf = header;

	spy_io_job_t *io_job;
	spy_chunk_t *chunk = NULL, *tmp;
	spy_file_index_entry_t *index_entry = NULL;

	assert(conn->body_len >= REQ_INNER_HDR_SIZE);
	body_len = conn->body_len - 8 - 2; // exclude fid + server_id

	n = spy_rw_buffer_read_n(&conn->request, buf, REQ_INNER_HDR_SIZE);
	assert (n == REQ_INNER_HDR_SIZE);

	server_id = spy_mach_read_from_2(buf);
	buf += 2;

	fid = spy_mach_read_from_8(buf);
	buf += 8;

	if (server_id != config.server_id) {
		conn->error = SERVER_ID_MISMATCH;
		goto ERROR_REPLY;
	}

	spy_lookup_file_index(fid, &index_entry, NULL);

	if (index_entry) {
		conn->error = FILE_EXISTS;
		goto ERROR_REPLY;
	}

	list_for_each_entry(tmp, &server.chunks, c_list) {
		if (tmp->avail_space > body_len + FILE_META_SIZE) {
			chunk = tmp;
			break;
		}
	}

	if (!chunk && server.pending_writes_size >= DEF_MAX_PENDING_WRITES) {
		conn->error = CHUNK_UNAVAILABLE;

		goto ERROR_REPLY;
	}

	//taken away from alloc list
	if (chunk)
		list_move(&chunk->c_list, &server.writing_chunks);

	io_job = spy_gen_io_job();
	if (!io_job) {
		conn->error = IO_JOB_UNAVAILABLE;
		goto ERROR_REPLY;
	}

	io_job->file.fid       = fid;
	io_job->file.ref       = 1;
	io_job->file.size      = body_len;

	io_job->conn           = conn;

	io_job->work.fn        = spy_write_file;
	io_job->work.done      = spy_write_file_done;

	spy_remove_file_event(server.event_loop, conn->fd, AE_READABLE);

	if (chunk) {
		// add to work queue

		statistic.write_count++;
		statistic.writing_count++;

		io_job->offset         = chunk->current_offset;
		io_job->chunk          = chunk;

		spy_queue_work(server.wq, &io_job->work);
	} else {
		// add to pending writes

		io_job->offset         = 0;
		io_job->chunk          = NULL;
		io_job->req_time       = time(NULL); // for timeout check

		list_add_tail(&io_job->pw_list, &server.pending_writes);
		server.pending_writes_size++;
	}

	return;

ERROR_REPLY:

	statistic.write_error++;

	conn->body_len = 0;
	conn->state    = SEND_RSP;

	spy_reply_header(conn);
}

static void spy_lookup_file_index(uint64_t fid, 
                                  spy_file_index_entry_t **index_entry,
                                  struct hlist_head **p_head)
{
	struct hlist_head *head;
	spy_file_index_entry_t *tmp;

	head = server.file_index + (fid % FILE_HASH_SIZE);
	hlist_for_each_entry(tmp, head, f_hash) {
		if (tmp->fid == fid) {
			*index_entry = tmp;
			break;
		}
	}

	if (p_head) {
		*p_head = head;
	}
}

static void spy_dump_chunkfile_done(spy_work_t *work)
{
	int retcode, sent_header;
	spy_dump_job_t *dump_job;
	spy_connection_t *conn;

	dump_job    = container_of(work, spy_dump_job_t, work);	
	conn        = dump_job->conn;
	retcode     = dump_job->retcode;

	free(dump_job);

	if (retcode != 0) {
		spy_free_client_connection(conn);
		return;
	}

	if (spy_set_blocking(conn->fd, 0)) {
		spy_log(ERROR, "dump chunk done, set socket nonblock failed, %s", strerror(errno));
		spy_free_client_connection(conn);

		return;
	}

	spy_reset_client_connection(conn);	
}

static void spy_dump_chunk(spy_connection_t *conn)
{
	int r;
	size_t n;
	spy_chunk_t *chunk = NULL, *tmp;
	spy_dump_job_t *dump_job = NULL;
	char buf[CHUNK_FILENAME_LEN_MAX + 1];

	dump_job = calloc(1, sizeof(spy_dump_job_t));
	assert(dump_job);

	if (conn->body_len >= CHUNK_FILENAME_LEN_MAX) {
		spy_log(ERROR, "request dump chunk filename length too large, %d", conn->body_len);
		conn->error    = INTERNAL_ERROR;
		goto ERROR_REPLY;
	}

	n = spy_rw_buffer_read_n(&conn->request, dump_job->chunk_name, conn->body_len);
	assert (n == conn->body_len);

	r = sprintf(buf, "/%s", dump_job->chunk_name);
	assert (r > 0);

	list_for_each_entry(tmp, &server.chunks, c_list) {
		if (spy_string_ends_with(tmp->path, buf)) {
			chunk = tmp;
			break;
		}
	}

	if (chunk == NULL) {
		conn->error = FILE_NOT_FOUND;
		goto ERROR_REPLY;
	}

	if (spy_set_blocking(conn->fd, 1)) {
		spy_log(ERROR, "dump chunk file, set socket block failed, %s", strerror(errno));
		conn->error    = INTERNAL_ERROR;
		goto ERROR_REPLY;
	}

	dump_job->conn      = conn;
	dump_job->chunk     = chunk;
	dump_job->work.fn   = spy_dump_chunkfile;
	dump_job->work.done = spy_dump_chunkfile_done;

	spy_remove_file_event(server.event_loop, conn->fd, AE_READABLE);
	spy_queue_work(server.wq, &dump_job->work);

	return;

ERROR_REPLY:
	conn->body_len = 0;
	conn->state    = SEND_RSP;

	spy_reply_header(conn);	
}

static void spy_handle_read(spy_connection_t *conn)
{
	uint16_t server_id;
	uint64_t fid;
	size_t n;

	char *body_data, *buf;
	char req_inner_hdr[REQ_INNER_HDR_SIZE];

	struct hlist_head *head;
	spy_io_job_t *io_job;
	spy_chunk_t *chunk, *ctmp;
	spy_file_index_entry_t *index_entry = NULL;

	buf = req_inner_hdr;
	n = spy_rw_buffer_read_n(&conn->request, buf, REQ_INNER_HDR_SIZE);
	assert (n == REQ_INNER_HDR_SIZE);

	server_id = spy_mach_read_from_2(buf);
	buf += 2;

	fid = spy_mach_read_from_8(buf);
	buf += 8;

	if (server_id != config.server_id) {
		conn->error = SERVER_ID_MISMATCH;
		goto ERROR_REPLY;
	}

	spy_lookup_file_index(fid, &index_entry, NULL);

	if (!index_entry) {
		conn->error = FILE_NOT_FOUND;
		goto ERROR_REPLY;
	}

	head = server.chunk_index + 
		(index_entry->chunk_id % CHUNK_HASH_SIZE);

	hlist_for_each_entry(ctmp, head, c_hash) {
		if (ctmp->chunk_id == index_entry->chunk_id) {
			chunk = ctmp;
			break;
		}
	}

	if (!chunk) {
		conn->error = FILE_NOT_FOUND;
		goto ERROR_REPLY;
	}

	while (conn->rsp_body.cap < index_entry->size) {
		if (spy_rw_buffer_expand(&conn->rsp_body) < 0) {
			spy_log(ERROR, "expand rsp_body failed. reach mem blocks limit.");

			conn->error = REACH_MEMORY_LIMIT;
			goto ERROR_REPLY;
		}
	}

	io_job = spy_gen_io_job();
	if (!io_job) {
		conn->error = IO_JOB_UNAVAILABLE;
		goto ERROR_REPLY;
	}

	io_job->index_entry     = index_entry;
	io_job->conn            = conn;
	io_job->chunk           = chunk;

	io_job->work.fn         = spy_read_file;
	io_job->work.done       = spy_read_file_done;

	statistic.read_count++;
	statistic.reading_count++;

	spy_remove_file_event(server.event_loop, conn->fd, AE_READABLE);
	spy_queue_work(server.wq, &io_job->work);

	return;

ERROR_REPLY:
	statistic.read_error++;

	conn->body_len = 0;
	conn->state    = SEND_RSP;

	spy_reply_header(conn);
}

static void spy_handle_check_disk(spy_connection_t *conn)
{
	// check disk health,
	// random read from a random chunk
	uint64_t chunk_id;
	struct hlist_head *head;
	spy_io_job_t *io_job;
	spy_chunk_t *chunk, *ctmp;

	chunk_id = rand() % server.n_chunks + 1;
	head = server.chunk_index + (chunk_id % CHUNK_HASH_SIZE);
	hlist_for_each_entry(ctmp, head, c_hash) {
		if (ctmp->chunk_id == chunk_id) {
			chunk = ctmp;
			break;
		}
	}
	assert (chunk);

	io_job = spy_gen_io_job();
	if (!io_job) {
		conn->error = IO_JOB_UNAVAILABLE;
		goto ERROR_REPLY;
	}

	io_job->conn           = conn;
	io_job->chunk          = chunk;

	io_job->work.fn        = spy_check_chunk;
	io_job->work.done      = spy_check_chunk_done;

	spy_remove_file_event(server.event_loop, conn->fd, AE_READABLE);
	spy_queue_work(server.wq, &io_job->work);

	return;

ERROR_REPLY:
	conn->body_len = 0;
	conn->state    = SEND_RSP;

	spy_reply_header(conn);
}

static void spy_process_receive_buffer(spy_connection_t *conn)
{
	size_t size;
	char header[REQ_HDR_SIZE];
	char *buf = header;

	if (conn->state == RECEIVE_HDR && 
			(conn->request.write_pos - conn->request.read_pos >= REQ_HDR_SIZE)) {

		size = spy_rw_buffer_read_n(&conn->request, buf, REQ_HDR_SIZE);
		assert (size == REQ_HDR_SIZE);

		conn->opcode = spy_mach_read_from_1(buf);
		buf += 1;

		conn->body_len = spy_mach_read_from_4(buf);

		conn->state = RECEIVE_BODY;
	}

	if (conn->state == RECEIVE_BODY &&
			(conn->request.write_pos - conn->request.read_pos >= conn->body_len)) {

		conn->state = PROCESS_REQUEST;
	}

	if (conn->state == PROCESS_REQUEST) {
		switch (conn->opcode) {
		case OPCODE_WRITE:
			spy_handle_write(conn);
			break;
		case OPCODE_READ:
			spy_handle_read(conn);
			break;

		case OPCODE_PING:
			spy_handle_ping(conn);
			break;

		case OPCODE_CHECK_DISK:
			spy_handle_check_disk(conn);
			break;

		case OPCODE_KILL_PD_WR:
			spy_kill_pending_writes(conn);
			break;

		case OPCODE_QUERY_IO_STATUS:
			spy_query_io_status(conn);
			break;

		case OPCODE_QUERY_DETAIL_INFOS:
			spy_query_detail_infos(conn);
			break;

		case OPCODE_DUMP_CHUNK:
			spy_dump_chunk(conn);
			break;

		default:
			spy_log(ERROR, "unknown client opcode %d",conn->opcode);
			spy_free_client_connection(conn);
		}
	}
}

static void spy_read_handler(aeEventLoop *el, int fd, void *priv, int mask)
{
	int nread;
	size_t len;
	char* buf;

	spy_connection_t *conn = (spy_connection_t *)priv;

	while (spy_rw_buffer_next_writeable(&conn->request, &buf, &len) < 0) {
		// no place for write, expand

		if (spy_rw_buffer_expand(&conn->request) < 0) {
			// expand failed, reach mem block limit
                        
			spy_log(ERROR, "expand request buffer failed");

			spy_free_client_connection(conn);

			return;
		}
	}

	nread = read(conn->fd, buf, len);

	if (nread < 0) {
		if (errno == EAGAIN)
			return;

		spy_log(ERROR, "read cli err: %s", strerror(errno));
		spy_free_client_connection(conn);
		return;
	}

	if (nread == 0) {
		spy_free_client_connection(conn);
		return;
	}

	conn->request.write_pos += nread;

	spy_process_receive_buffer(conn);
}

static void spy_create_client_connection(int fd)
{
	int n;
	spy_connection_t *conn = calloc(1, sizeof(spy_connection_t));

	if (!conn) {
		spy_log(ERROR, "create connection failed");
		return;
	}

	memset(conn, 0, sizeof(spy_connection_t));
	conn->fd        = fd;
	conn->state     = RECEIVE_HDR;
	spy_rw_buffer_init(&conn->request);
	spy_rw_buffer_init(&conn->rsp_body);

	n = spy_create_file_event(server.event_loop, fd,
                              AE_READABLE, spy_read_handler, conn);

	if (n == AE_ERR) {
		spy_log(ERROR, "create read event failed");

		free(conn);
		return;
	}

	statistic.conn_count++;
}

static void spy_wq_thread_done(aeEventLoop *el, int fd, void *priv, int mask)
{
	uint64_t dummy;
	spy_work_t *work;
	LIST_HEAD(list);

	spy_work_queue_t *wq = priv;

	read((server.wq)->finished_event_fd, &dummy, 8);
	
	pthread_mutex_lock(&wq->finished_lock);
	list_splice_init(&wq->finished_list, &list);
	pthread_mutex_unlock(&wq->finished_lock);
	
	while (!list_empty(&list)) {
		work = list_first_entry(&list, spy_work_t, w_list);
		list_del(&work->w_list);
		
		work->done(work);
	}
}

static void spy_collect_free_space(uint64_t *total, uint64_t *max)
{
	spy_chunk_t *tmp;
        
	*total  = 0;
	*max    = 0;

	list_for_each_entry(tmp, &server.chunks, c_list) {
		*total += tmp->avail_space;

		if (tmp->avail_space > *max)
			*max = tmp->avail_space;
	}

	list_for_each_entry(tmp, &server.writing_chunks, c_list) {
		*total += tmp->avail_space;

		if (tmp->avail_space > *max)
			*max = tmp->avail_space;
	}
}

static void spy_report_server_infos()
{
	uint64_t total_free_space, max_free_space;

	if (!spy_atomic_cmp_and_set(&report_info.lock, 0, 1)) {
		spy_log(INFO, "someone else is operating the report info.");
		return;
	}

	spy_collect_free_space(&total_free_space, &max_free_space);

	report_info.max_free_space   = max_free_space;
	report_info.total_free_space = total_free_space;

	report_info.status           = server.status;
	report_info.n_chunks         = server.n_chunks;
	report_info.pending_writes   = server.pending_writes_size;
	report_info.writing_count    = statistic.writing_count;
	report_info.reading_count    = statistic.reading_count;
	report_info.conn_count       = statistic.conn_count;
	
	spy_atomic_cmp_and_set(&report_info.lock, 1, 0);

	return;
}

static void spy_remove_timeout_pending_writes()
{
	spy_io_job_t *pending_io_job, *tmp;
	spy_connection_t *conn;
	time_t now;

	if (server.pending_writes_size == 0)
		return;

	assert (!list_empty(&server.pending_writes));

	now = time(NULL);

	list_for_each_entry_safe(pending_io_job, tmp, &server.pending_writes, pw_list) {

		// we add pending writes to tail, and iterate from head.
		if (pending_io_job->req_time + DEF_PENDING_WRITE_TIMEOUT > now)
			break;

		list_del(&pending_io_job->pw_list);
		server.pending_writes_size --;

		conn = pending_io_job->conn;

		conn->error    = CHUNK_UNAVAILABLE;
		conn->body_len = 0;
		conn->state    = SEND_RSP;

		spy_reply_header(conn);

		spy_free_io_job(pending_io_job);
	}
}

static int spy_server_cron(aeEventLoop *el, long long id, void *arg)
{
	spy_report_server_infos();

	spy_remove_timeout_pending_writes();

	// next cycle 1 seconds later
	return 1000;
}

static void spy_accept_handler(aeEventLoop *el, int fd, void *priv, int mask)
{
	int cfd;
	struct sockaddr_in caddr;	
	socklen_t caddr_len = sizeof(struct sockaddr_in);
	
	cfd = accept4(fd, (struct sockaddr *)&caddr, &caddr_len, SOCK_NONBLOCK);
	if (cfd < 0) {
		spy_log(ERROR, "accept connection failed: %s", strerror(errno));
		return;
	}
	
	spy_create_client_connection(cfd);
}

static int spy_set_blocking(int fd, int block)
{
    int flags;

    if ((flags = fcntl(fd, F_GETFL)) == -1) {
        return -1;
    }

    if (block) {
		flags &= ~O_NONBLOCK;
	} else {
        flags |= O_NONBLOCK;
	}

    if (fcntl(fd, F_SETFL, flags) == -1) {
        return -1;
    }

    return 0;
}

static int spy_create_listen_socket(int port, char *bind_addr)
{
	int fd, on = 1;
	struct sockaddr_in addr;

	if ((fd = socket(AF_INET, SOCK_STREAM, 0)) == -1) {
		spy_log(ERROR, "create socket failed, %s", strerror(errno));
		return -1;
	}

	if ((setsockopt(fd, SOL_SOCKET, SO_REUSEADDR, &on, sizeof(on))) == -1) {
		spy_log(ERROR, "set reuse addr failed, %s", strerror(errno));
		return -1;
	}

	memset(&addr, 0, sizeof(addr));
	addr.sin_family = AF_INET;
	addr.sin_port = htons(port);
	addr.sin_addr.s_addr = htonl(INADDR_ANY);

	if (bind_addr && inet_aton(bind_addr, &addr.sin_addr) == 0) {
		spy_log(ERROR, "bind sockaddr %s failed, invalid ip address", bind_addr);
		close(fd);
		return -1;
	} 

	if (bind(fd, (struct sockaddr *)&addr, sizeof(addr))) {
		spy_log(ERROR, "bind socket failed, error %s", strerror(errno));
		close(fd);
		return -1;
	}

	return fd;
}

static void spy_setup_work_queue()
{
	int n;
	
	server.wq = spy_create_work_queue(DEF_WQ_NR_THRS);
	if (!server.wq) {
		spy_log(ERROR, "create work queue failed");
		exit(1);
	}

	n = spy_create_file_event(server.event_loop, 
						  (server.wq)->finished_event_fd, AE_READABLE,
						  spy_wq_thread_done, server.wq);

	if (n == AE_ERR) {
		spy_log(ERROR, "create eventfd event failed");
		exit(1);
	}
}

static void spy_setup_listen_events(int listen_fd)
{
	int n;
	
	if (listen(listen_fd, 512)) {
		spy_log(ERROR, "listen socket failed, error %s", strerror(errno));
		exit(1);
	}

	server.listen_fd  = listen_fd;
	server.event_loop = spy_create_event_loop(DEF_MAX_CLIENTS);

	if (!server.event_loop) {
		spy_log(ERROR, "create event loop failed");
		exit(1);
	}

	n = spy_create_file_event(server.event_loop, server.listen_fd, 
							  AE_READABLE, spy_accept_handler, NULL);
	
	if (n == AE_ERR) {
		spy_log(ERROR, "create file event failed");
		exit(1);
	}

	n = spy_create_time_event(server.event_loop, 1000, spy_server_cron, 
							  NULL, NULL);

	if (n == AE_ERR) {
		spy_log(ERROR, "create time event failed");
		exit(1);
	}
}

static uint32_t spy_get_bind_addr()
{
	struct in_addr in;
	uint32_t ipnum = 0;

	if (config.bind_addr) {
		inet_aton(config.bind_addr, &in);
		ipnum = in.s_addr;
	}

	return ipnum;
}

static void spy_setup_mem_blocks()
{
	// init mem blocks && prealloc memory
	init_mem_blocks(&server.free_mem_blocks, config.mb_prealloc_count);

	server.mem_blocks_alloc         = config.mb_prealloc_count;
	server.mem_blocks_alloc_limit   = config.mb_limit;
}

static void spy_setup_io_jobs()
{
	spy_init_io_jobs(DEF_MAX_IO_JOBS);
}

static void spy_usage()
{
	printf("usage:spy_server\n"
		   "===========================\n"
		   "[--port=<port>]\n"
		   "[--ip=<listen address>]\n"
		   "[--data_dir=<data directory>]\n"
		   "[--error_log=<error log file>]\n"
		   "[--mem_blocks=<memory blocks for stream buffer>]\n"
		   "[--chunks=<number of chunks>]\n"
		   "[--sync=<sync when write, 1 or 0>]\n"
		   "[--daemonize=<1 or 0>]\n"
		   "--master_ip=<chunkmaster ip addr>\n"
		   "--master_port=<chunkmaster port>\n"
		   "--group_id=<unique group id>\n"
		);

	exit(1);
}

static void spy_open_log_file()
{
	server.log_fd = open(config.log_path, O_WRONLY | O_CREAT, 0644);

	if (server.log_fd <= 0) {
		printf("open log file failed\n");
		exit(1);
	}
}

static void spy_server_init() 
{
	memset(&config, 0x0, sizeof(spy_config_t));

	config.port       = DEF_SERVER_PORT;
	config.daemonize  = DEF_DAEMONIZE;
	config.data_dir   = DEF_DATA_DIR;
	config.log_path   = DEF_LOG_PATH; 
	config.chunk_size = DEF_CHUNK_SIZE;
	config.n_chunks   = DEF_N_CHUNKS;
	config.log_level  = DEF_LOG_LEVEL;
	config.bind_addr  = DEF_SERVER_ADDR;
	config.mb_prealloc_count = PREALLOC_COUNT;
	config.mb_limit   = DEF_MEM_BLOCKS_LIMIT;
	config.sync       = 0;
	config.server_id  = -1;

	memset(&server, 0x0, sizeof(server));

	INIT_LIST_HEAD(&server.chunks);
	INIT_LIST_HEAD(&server.writing_chunks);
	INIT_LIST_HEAD(&server.free_mem_blocks);
	INIT_LIST_HEAD(&server.pending_writes);
	INIT_LIST_HEAD(&server.free_io_jobs);
	server.status = STATUS_RW;

	memset(&statistic, 0x0, sizeof(statistic));

	// init rand
	srand(time(NULL));
}

static void spy_sigterm_handler(int sig)
{
	//TODO: clean shutdown
}

static void spy_signal_init()
{
	struct sigaction act;

	signal(SIGHUP, SIG_IGN);
	signal(SIGPIPE, SIG_IGN);

	sigemptyset(&act.sa_mask);
	act.sa_flags = SA_RESETHAND | SA_ONSTACK | SA_NODEFER;
	act.sa_handler = spy_sigterm_handler;
	sigaction(SIGTERM, &act, NULL);
}

void spy_check_server_options()
{

	int error = 0;

	error = (config.master_addr == NULL ||
			 config.master_port == 0    ||
			 config.server_id == -1);

	if (error) {
		spy_usage();
		exit(1);
	}

}

int main(int argc, char **argv)
{
	int c, n, listen_fd;
	char *p;

	spy_signal_init();
	spy_server_init();

	while ((c = getopt_long(argc, argv, server_options, server_long_options, NULL)) >= 0) {
		switch (c) {
		case 'p':
			config.port = strtol(optarg, &p, 10);
			if (config.port <= 0 || config.port > UINT16_MAX ||
				*p != '\0') {
				printf("invalid listen port\n");
				exit(1);
			}
			break;
		case 'r':
			config.master_port = strtol(optarg, &p, 10);
			if (config.master_port <= 0 || config.master_port > UINT16_MAX ||
				*p != '\0') {
				printf("invalid master port\n");
				exit(1);
			}
			break;
		case 'h':
			config.bind_addr = strdup(optarg);
			if (!config.bind_addr) {
				printf("dup bind_addr failed, not enough memory\n");
				exit(1);
			}
			break;
		case 'm':
			config.master_addr = strdup(optarg);
			if (!config.master_addr) {
				printf("dup master_addr failed, not enough memory\n");
				exit(1);
			}
			break;
		case 'w':
			config.data_dir = strdup(optarg);
			if (!config.data_dir) {
				printf("dup data_dir failed, not enough memory\n");
				exit(1);
			}
			break;
		case 'd':
			config.daemonize = 1;
			break;
		case 'e':
			config.log_path = strdup(optarg);
			if (!config.log_path) {
				printf("dup log_path failed, not enough memory\n");
				exit(1);
			}
			break;
		case 'g':
			config.server_id = strtol(optarg, &p, 10);
			if (config.server_id <= 0 || config.server_id > UINT16_MAX ||
				*p != '\0') {
				printf("invalid server id\n");
				exit(1);
			}
			break;
		case 'b':
			config.mb_limit = strtol(optarg, &p, 10);
			if (config.mb_limit <= 0 || config.mb_limit > MAX_MEM_BLOCKS_LIMIT) {
				printf("invalid memory block size\n");
				exit(1);
			}
			break;
		case 'c':
			config.n_chunks = strtol(optarg, &p, 10);
			if (config.n_chunks <= 0 || config.n_chunks > MAX_N_CHUNKS) {
				printf("invalid n_chunks\n");
				exit(1);
			}
			break;
		case 'f':
			config.sync = strtol(optarg, &p, 10);
			if (config.sync != 1 && config.sync != 0) {
				printf("invalid sync arguments\n");
				exit(1);
			}
			break;
		default:
			spy_usage();
		}
	}

	spy_check_server_options();

	spy_open_log_file();

	if (config.daemonize) {
		spy_make_daemonize();
	}

	spy_log(INFO, "server starting...");

	listen_fd = spy_create_listen_socket(config.port, config.bind_addr);

	if (listen_fd < 0) {
		spy_log(ERROR, "create listen socket failed: %s", strerror(errno));
		exit(1);
	}

	spy_create_or_recover_files(config.data_dir);

	spy_setup_listen_events(listen_fd);

	spy_setup_work_queue();

	spy_setup_mem_blocks();

	spy_setup_io_jobs();

	spy_start_agent_thread();

	spy_log(INFO, "server start successful!");

	spy_start_process_events(server.event_loop);
}
