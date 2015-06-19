#include <sys/types.h>
#include <dirent.h>
#include <string.h>
#include <errno.h>
#include <stdlib.h>
#include <assert.h>
#include <fcntl.h>
#include <unistd.h>
#include <stdio.h>
#include <stdarg.h>
#include <stddef.h>
#include <sys/stat.h>
#include <sys/statvfs.h>
#include <inttypes.h>

#include "spy_server.h"
#include "spy_store.h"
#include "spy_utils.h"
#include "spy_adler32.h"
#include "spy_list.h"
#include "spy_log.h"
#include "spy_utils.h"
#include "spy_rw_buffer.h"

static spy_chunk_t *spy_alloc_chunk(int fd, uint64_t chunk_id, char *path, uint32_t size);

static void spy_add_chunk(spy_chunk_t *chunk)
{
	spy_chunk_t *tmp;
	struct hlist_head *head;

	list_add(&chunk->c_list, &server.chunks);

	head = server.chunk_index + 
		(chunk->chunk_id % CHUNK_HASH_SIZE);

	hlist_for_each_entry(tmp, head, c_hash) {
		assert(tmp->chunk_id != chunk->chunk_id);
	}

	hlist_add_head(&chunk->c_hash, head);

	server.n_chunks++;
}

static void spy_add_file_index(uint64_t chunk_id, uint64_t fid, 
                               uint64_t offset,uint32_t size)
{
	spy_file_index_entry_t *index_entry, *tmp;
	struct hlist_head *head;

	index_entry = malloc(sizeof(spy_file_index_entry_t));
	assert(index_entry);

	index_entry->chunk_id = chunk_id;
	index_entry->fid      = fid;
	index_entry->offset   = offset;
	index_entry->size     = size;

	head = server.file_index + (fid % FILE_HASH_SIZE);
	hlist_for_each_entry(tmp, head, f_hash) {
		assert(tmp->fid != fid);
	}

	hlist_add_head(&index_entry->f_hash, head);
}

static void spy_parse_data_block(uint64_t chunk_id, int base_pos,
								 char *buf, int *plen)
{
	char *p = buf;
	int len = *plen, parsed, left;
	uint64_t fid, offset, checksum, foff;
	uint32_t size;

	assert(len >= FILE_META_SIZE);

	while (1) {
		fid = spy_mach_read_from_8(p);
		p += 8;
		
		//skip ref
		p += 2;
		
		size = spy_mach_read_from_4(p);
		p += 4;
		
		checksum = spy_mach_read_from_8(p);
		p += 8;
		
		//skip ts
		p += 8;

		if ((len - (p - buf)) < size) {
			p -= FILE_META_SIZE;
			break;
		}

		//do index recovery
		foff = base_pos + (p - buf) - FILE_META_SIZE;
		spy_add_file_index(chunk_id, fid, foff, size + FILE_META_SIZE);	
	
		//TODO: read data & do checksum check
		p += size;

		if ((len - (p - buf)) < FILE_META_SIZE) 
			break;
	}

	parsed = p - buf;
	left   = len - parsed;
	
	if (left > 0) {
		memmove(buf, p, left);
		*plen = left;
	}
}

static spy_file_parser_t *spy_alloc_file_parser(int fd, spy_chunk_t *chunk, 
                                                int fpos, size_t buffer_size)
{
	spy_file_parser_t *parser;

	parser = malloc(sizeof(spy_file_parser_t));
	assert(parser);

	parser->fd           = fd;
	parser->fpos         = fpos;
	parser->chunk_id     = chunk->chunk_id;
	parser->chunk_offset = chunk->current_offset;
	parser->buf          = malloc(buffer_size);

	assert(parser->buf);

	parser->buf_size     = buffer_size;
	parser->buf_pos      = 0;
	parser->buf_left     = 0;

	return parser;
}

static void spy_free_file_parser(spy_file_parser_t *parser)
{
	if (parser) {
		if (parser->buf)
			free(parser->buf);
		free(parser);
	}
}

static int spy_parse_next_file(spy_file_parser_t *parser)
{
	int fleft, cur_fleft, buf_free;
	size_t nread, rsize, size;
	uint64_t fid, checksum, foff;
	char *p;

	// no more files
	if (parser->fpos >= parser->chunk_offset && parser->buf_left == 0)
		return 0;

	// read data to buffer if there is not enough data
	if (parser->buf_left < FILE_META_SIZE) {
		if (parser->buf_left > 0)
			memmove(parser->buf, parser->buf + parser->buf_pos, parser->buf_left);

		parser->buf_pos = 0;

		fleft = parser->chunk_offset - parser->fpos;
		if (fleft <= 0) return -1;

		buf_free = parser->buf_size - parser->buf_left;
		rsize    = fleft < buf_free ? fleft : buf_free;

		do{
			nread = read(parser->fd, parser->buf + parser->buf_left, rsize);
		} while (nread < 0 && errno == EINTR);
        
		if (nread <= 0) return -1;

		parser->buf_left += nread;
		parser->fpos += nread;
	}

	// parse file header
	assert(parser->buf_left >= FILE_META_SIZE);
	p = parser->buf + parser->buf_pos;
    
	fid = spy_mach_read_from_8(p);
	p += 8;

	//skip ref
	p += 2;
		
	size = spy_mach_read_from_4(p);
	p += 4;
		
	checksum = spy_mach_read_from_8(p);
	p += 8;
		
	//skip ts
	p += 8;

	parser->buf_pos += FILE_META_SIZE;
	parser->buf_left -= FILE_META_SIZE;

	// parse file body
	// TODO: calc checksum
	if (parser->buf_left < size) {
        
		cur_fleft = size - parser->buf_left;

		while (cur_fleft > 0) {

			fleft = parser->chunk_offset - parser->fpos;
			if (fleft <= 0) return -1;

			rsize = fleft < parser->buf_size ? fleft : parser->buf_size;
			nread = read(parser->fd, parser->buf, rsize);
            
			if (nread < 0 && errno == EINTR)
				continue;
			if (nread <= 0)
				return -1;

			parser->fpos    += nread;
			parser->buf_left = nread;

			if (cur_fleft > nread) {
				cur_fleft       -= nread;
				parser->buf_pos  = 0;
				parser->buf_left = 0;
			} else {
				parser->buf_pos   = cur_fleft;
				parser->buf_left -= cur_fleft;
				cur_fleft         = 0;
			}
		}

	} else {
		parser->buf_pos  += size;
		parser->buf_left -= size;
	}

	// add memory index
	foff = parser->fpos - size - FILE_META_SIZE - parser->buf_left;
	spy_add_file_index(parser->chunk_id, fid, foff, size + FILE_META_SIZE);

	return 1;
}

static void spy_parse_chunk_superblock(char *sb_buf, spy_chunk_t *chunk)
{
	char *p = sb_buf;
	uint64_t checksum, checksum2;

	assert(!memcmp(p, SUPERBLOCK_MAGIC, SUPERBLOCK_MAGIC_SIZE));
	p += SUPERBLOCK_MAGIC_SIZE;
	
	chunk->chunk_id = spy_mach_read_from_8(p);
	p += 8;
	
	chunk->files_alloc = spy_mach_read_from_4(p);
	p += 4;

	chunk->files_count = spy_mach_read_from_4(p);
	p += 4;

	chunk->current_offset = spy_mach_read_from_8(p);
	p += 8;	

	chunk->avail_space = spy_mach_read_from_8(p);
	p += 8;	

	checksum  = spy_mach_read_from_8(p);
	checksum2 = (uint64_t)spy_adler32(0UL, (unsigned char *)sb_buf, p - sb_buf);

	assert(checksum == checksum2);

	if (server.max_chunk_id < chunk->chunk_id) {
		server.max_chunk_id = chunk->chunk_id;
	}
}

static void spy_setup_chunk(char *chunk_path)
{
	int fd, nread, result, fpos = 0, in = 0, complete = 0;
	char sb_buf[SUPERBLOCK_SIZE], *block;
	spy_chunk_t *chunk;
	struct stat st;
	spy_file_parser_t *parser;

	/*
	 * 1. create spy_chunk_t struct,add it to chunks list
	 * 2. build in-memory file index
	 */
	fd = open(chunk_path, O_RDWR, 0644);
	assert(fd);

	assert(fstat(fd, &st) == 0);

	chunk = spy_alloc_chunk(fd, 0, chunk_path, st.st_size);

	do {
		nread = read(fd, sb_buf, SUPERBLOCK_SIZE);
	} while (nread < 0 && errno == EINTR);

	if (nread <= 0) {
		spy_log(ERROR, "read chunk file %s superblock failed: %s\n", 
			   chunk_path, strerror(errno));
		exit(1);
	}

	spy_parse_chunk_superblock(sb_buf, chunk);
	fpos += SUPERBLOCK_SIZE;

	parser = spy_alloc_file_parser(fd, chunk, fpos, BUFFER_SIZE);

	while (result = spy_parse_next_file(parser)) {
		if (result < 0) {
			spy_log(ERROR, "parse next file failed\n");
			break;
		}
	}
    
	fpos = parser->fpos;
	spy_free_file_parser(parser);

	if (fpos < chunk->current_offset) {
		spy_log(ERROR, "chunk file read incomplete\n");
		exit(1);
	}

	spy_add_chunk(chunk);
}

static spy_chunk_t *spy_alloc_chunk(int fd, uint64_t chunk_id, char *path, uint32_t size)
{
	spy_chunk_t *chunk;

	chunk = malloc(sizeof(spy_chunk_t));
	assert(chunk);

	chunk->fd             = fd;
	chunk->path           = strdup(path);
	chunk->chunk_id       = chunk_id;
	chunk->files_alloc    = 0;
	chunk->files_count    = 0;
	chunk->file_size      = size;
	chunk->avail_space    = config.chunk_size - SUPERBLOCK_SIZE;
	chunk->current_offset = SUPERBLOCK_SIZE;
	
	INIT_LIST_HEAD(&chunk->c_list);

	return chunk;
}

static void spy_fill_superblock(char *buf, spy_chunk_t *chunk)
{
	char *p = buf;
	uint64_t checksum;

	memcpy(p, SUPERBLOCK_MAGIC, SUPERBLOCK_MAGIC_SIZE);
	p += SUPERBLOCK_MAGIC_SIZE;

	spy_mach_write_to_8(p, chunk->chunk_id);
	p += 8;

	spy_mach_write_to_4(p, chunk->files_alloc);
	p += 4;

	spy_mach_write_to_4(p, chunk->files_count);
	p += 4;

	spy_mach_write_to_8(p, (uint64_t)chunk->current_offset);
	p += 8;	

	spy_mach_write_to_8(p, (uint64_t)chunk->avail_space);
	p += 8;	

	checksum = (uint64_t)spy_adler32(0UL, (unsigned char *)buf, p - buf);
	spy_mach_write_to_8(p, checksum);
}

//TODO: need double write buffer here
int spy_write_chunk_superblock(spy_chunk_t *chunk)
{
	ssize_t n;
	char buf[SUPERBLOCK_SIZE];

	spy_fill_superblock(buf, chunk);
	
	n = spy_pwrite(chunk->fd, buf, SUPERBLOCK_SIZE, 0);
	
	if (n != SUPERBLOCK_SIZE) {
		spy_log(ERROR, "write superblock failed: %s", strerror(errno));
		return -1;
	}

	return 0;
}

int spy_flush_and_sync_chunk(spy_chunk_t *chunk)
{
	return fsync(chunk->fd);
}

void WORKER_FN spy_dump_chunkfile(spy_work_t *wk)
{
	
	off_t offset = 0;
	unsigned char header[6];
	spy_chunk_t *chunk;
	spy_connection_t *conn;
	spy_dump_job_t *dump_job;
	ssize_t n, sent = 0, total = 6;

	dump_job = container_of(wk, spy_dump_job_t, work);
	conn     = dump_job->conn;
	chunk    = dump_job->chunk;

	dump_job->retcode = 0;

	spy_mach_write_to_1(conn->rsp_header, conn->opcode);
	spy_mach_write_to_1(conn->rsp_header + 1, 0);
	spy_mach_write_to_4(conn->rsp_header + 2, chunk->file_size);

	while (total > 0) {
		n = write(conn->fd, conn->rsp_header + sent, total - sent);
		if (n < 0) {
			dump_job->retcode = -1;
			return;
		}

		sent  += n;
		total -= n;
	}

	while (offset != chunk->file_size) {   
		n = sendfile64(conn->fd, chunk->fd, &offset, chunk->file_size - offset);
		if (n < 0) {
			spy_log(ERROR, "sendfile error, err:%s", strerror(errno));
			dump_job->retcode = -1;		
			return;
		}
	}
}

/*
 * check disk health.
 * random read a block from file.
 */
void WORKER_FN spy_check_chunk(spy_work_t *wk)
{
	int n;
	char buf[DEF_CHECK_BLOCK_SIZE];
	uint64_t offset;

	spy_io_job_t *io_job = container_of(wk, spy_io_job_t, work);
	spy_chunk_t *chunk   = io_job->chunk;

	offset  = rand() % (DEF_CHUNK_SIZE / DEF_CHECK_BLOCK_SIZE);
	offset *= DEF_CHECK_BLOCK_SIZE;

	n = spy_pread(chunk->fd, buf, DEF_CHECK_BLOCK_SIZE, offset);

	io_job->retcode = (n == DEF_CHECK_BLOCK_SIZE) ? 0 : -1;
}

void WORKER_FN spy_read_file(spy_work_t *wk)
{
	int n;
	size_t left, shouldread, len;
	char *buf, filemeta[FILE_META_SIZE];
	uint64_t checksum, offset;

	spy_file_t *file;
	spy_chunk_t *chunk;
	spy_file_index_entry_t *index;
	spy_connection_t *conn;
	spy_io_job_t *io_job = container_of(wk, spy_io_job_t, work);

	conn    = io_job->conn;
	chunk   = io_job->chunk;
	index   = io_job->index_entry;
	file    = &io_job->file;
	offset  = index->offset;

	assert (conn->rsp_body.cap >= index->size);

	left = index->size;
	while (left > 0) {
		// must == 0
		assert (spy_rw_buffer_next_writeable(&conn->rsp_body, &buf, &len) == 0);

		shouldread = MIN(left, len);

		n = spy_pread(chunk->fd, buf, shouldread, offset);
		if (n != shouldread) {
			io_job->retcode = -1;
			return;
		}

		left    -= n;
		offset  += n;

		conn->rsp_body.write_pos += n;
	}

	n = spy_rw_buffer_read_n(&conn->rsp_body, filemeta, FILE_META_SIZE);
	assert (n == FILE_META_SIZE);

	buf = filemeta;

	file->fid = spy_mach_read_from_8(buf);
	buf += 8;

	file->ref = spy_mach_read_from_2(buf);
	buf += 2;

	file->size = spy_mach_read_from_4(buf);
	buf += 4;

	file->checksum = spy_mach_read_from_8(buf);
	buf += 8;

	file->timestamp = spy_mach_read_from_8(buf);
	buf += 8;

	checksum = (uint64_t)spy_buffer_adler32(0UL, 
						&conn->rsp_body, index->size - FILE_META_SIZE);

	if (file->fid != index->fid || 
		file->checksum != checksum) {
		io_job->retcode = -1;
		return;
	}

	io_job->retcode = 0;
}

static int spy_pwrite_file(int fd, char *data, size_t count, uint64_t offset)
{
	char *p = data;	
	ssize_t total = 0;

	while (count > 0) {
		ssize_t written = pwrite(fd, p, count, offset);
		if (written < 0 && (errno == EAGAIN || errno == EINTR))
			continue;
		if (written <= 0) {
			return -1;
		}

		count -= written;
		p += written;
		total += written;
		offset += written;
	}

	return total;
}

void spy_fill_file_meta(spy_io_job_t *io_job, char *meta, size_t len)
{
	char *buf               = meta;
	spy_file_t *file        = &io_job->file;
	spy_connection_t *conn  = io_job->conn;

	assert (len == FILE_META_SIZE);

	spy_mach_write_to_8(buf, file->fid);
	buf += 8;

	spy_mach_write_to_2(buf, file->ref);
	buf += 2;

	spy_mach_write_to_4(buf, file->size);
	buf += 4;

	file->checksum = 
		(uint64_t)spy_buffer_adler32(0UL, &conn->request, file->size);
	spy_mach_write_to_8(buf, file->checksum);
	buf += 8;

	spy_mach_write_to_8(buf, spy_current_time_sec());
	buf += 8;
}

void WORKER_FN spy_write_file(spy_work_t *wk)
{
	int nwrite, shouldwrite, left, n;
	size_t len;
	uint64_t offset;
	char filemeta[FILE_META_SIZE];
	char *buf;

	spy_io_job_t *io_job    = container_of(wk, spy_io_job_t, work);
	spy_connection_t *conn  = io_job->conn;
	spy_chunk_t *chunk      = io_job->chunk;
	offset                  = io_job->offset;

	spy_fill_file_meta(io_job, filemeta, FILE_META_SIZE);

	// write header
	// TODO: merge little file's write header && write body
	nwrite = spy_pwrite(chunk->fd, filemeta, FILE_META_SIZE, offset);
	if (nwrite != FILE_META_SIZE) {
		spy_log(ERROR, "write file meta failed");

		goto WRITE_ERROR;
	}
	offset += FILE_META_SIZE;

	// write body
	left = io_job->file.size;
	while (left > 0) {
		if (spy_rw_buffer_next_readable(&conn->request, &buf, &len) < 0)
			break;

		shouldwrite = MIN(len, left);
		nwrite = spy_pwrite(chunk->fd, buf, shouldwrite, offset);
		if (nwrite != shouldwrite) {
			spy_log(ERROR, "write file body failed");

			goto WRITE_ERROR;
		}

		offset  += nwrite;
		left    -= nwrite;
		conn->request.read_pos += nwrite;
	}

	if (left != 0)
		goto WRITE_ERROR;

	io_job->retcode = 0;

	chunk->current_offset += io_job->file.size + FILE_META_SIZE;
	chunk->avail_space    -= io_job->file.size + FILE_META_SIZE;
	chunk->files_alloc++;
	chunk->files_count++;
	
	n = spy_write_chunk_superblock(chunk);

	if (n) {
		//TODO: need double write buffer to avoid this
		spy_log(ERROR, "sb write error, chunk file maybe broken");		
	}
	
	if (!n && config.sync) {
		n = spy_flush_and_sync_chunk(chunk);
		if (n) {
			spy_log(ERROR, "sb write fsync failed, err %s", strerror(errno));
		}
	}

	return;

WRITE_ERROR:
	io_job->retcode = -1;

	return;
}

int spy_create_chunk()
{
	int fd, n;
	uint64_t chunk_id;
	char chunk_path[512];
	spy_chunk_t *chunk = NULL, *tmp;

	chunk_id = ++server.max_chunk_id;
	n = snprintf(chunk_path, 512, "%s/chunk_%d_%"PRIu64".chunk", 
                 config.data_dir, config.server_id, chunk_id);
	assert(n < 512);

	fd = open(chunk_path, O_CREAT | O_RDWR | O_LARGEFILE, 0644);
	if (fd < 0) {
		spy_log(ERROR, "create chunk file failed: %s", strerror(errno));
		return -1;
	}

	//n = posix_fallocate(fd, 0, config.chunk_size);
	n = fallocate(fd, 0, 0, config.chunk_size);
	if (n) {
		spy_log(ERROR, "fallocate chunk file failed: %s", strerror(errno));
		return -1;
	}

	chunk = spy_alloc_chunk(fd, chunk_id, chunk_path, config.chunk_size);
	spy_add_chunk(chunk);

	return spy_write_chunk_superblock(chunk);
}

static int spy_check_filename_format(char *filename)
{
	int server_id, n;
	uint64_t chunk_id;
	char *pattern = "chunk_%d_%"PRIu64".chunk";

	n = sscanf(filename, pattern, &server_id, &chunk_id);

	if (n != 2 || server_id != config.server_id) {
		return -1;
	}

	return 0;
}

void spy_create_or_recover_files(char *dir)
{
	int n, i;
	DIR *d;
	uint64_t free_space;
	char chunk_path[512];
	struct statvfs stat;
	struct dirent *file;
	
	d = opendir(dir);

	if (!d) {
		spy_log(ERROR, "open work dir %s failed:%s\n", dir, 
				strerror(errno));
		exit(1);
	}

	while ((file = readdir(d)) != NULL) {
		if (file->d_type == DT_REG) {
			if (spy_check_filename_format(file->d_name)) {
				spy_log(ERROR, "data directory contains invalid file or group id mismatch");
				exit(1);
			}

			n = snprintf(chunk_path, 512, "%s/%s", dir, file->d_name);
			assert(n < 512);

			spy_setup_chunk(chunk_path);
		}
	}

	closedir(d);

	if (list_empty(&server.chunks)) {
		for (i = 0; i < config.n_chunks; i++) {
			n = spy_create_chunk();
			if (n != 0) {
				spy_log(ERROR, "create new chunk file failed\n");
				exit(1);
			}
		}
	}
}
