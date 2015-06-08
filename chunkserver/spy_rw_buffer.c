#include <string.h>
#include <assert.h>

#include "spy_rw_buffer.h"
#include "spy_server.h"


void spy_rw_buffer_init(spy_rw_buffer_t *buffer)
{
	assert (buffer);

	INIT_LIST_HEAD(&buffer->mem_blocks);

	buffer->read_pos        = 0;
	buffer->write_pos       = 0;
	buffer->cap             = 0;

	buffer->read_block      = NULL;
	buffer->write_block     = NULL;
	buffer->read_base       = 0;
	buffer->write_base      = 0;
}

int spy_rw_buffer_expand(spy_rw_buffer_t *buffer)
{
	spy_mem_block_t *mem_block = NULL;

	if (list_empty(&server.free_mem_blocks)) {
		if (server.mem_blocks_alloc >= server.mem_blocks_alloc_limit) {
			spy_log(ERROR, "mem block reach limit");
			return -1;
		}

		mem_block = calloc(1, MEM_BLOCK_SIZE);
		if (!mem_block) {
			spy_log(ERROR, "calloc mem block failed");
			return -1;
		}

		INIT_LIST_HEAD(&mem_block->list);
		mem_block->size = MEM_BLOCK_SIZE - sizeof(spy_mem_block_t);
		
		server.mem_blocks_alloc++;
	} else {
		mem_block = list_first_entry(&server.free_mem_blocks,
                                            spy_mem_block_t, list);
	
		list_del_init(&mem_block->list);	
	}

	assert(mem_block);

	// first mem block
	if (list_empty(&buffer->mem_blocks)) {
		assert(!buffer->read_block && !buffer->write_block);

		buffer->read_block = mem_block;
		buffer->write_block = mem_block;
	}

	list_add_tail(&mem_block->list, &buffer->mem_blocks);
	buffer->cap += mem_block->size;

	server.mem_blocks_used++;

	return 0;
}

void spy_rw_buffer_reset(spy_rw_buffer_t *buffer)
{
	spy_mem_block_t *mem_block;

	assert(buffer);

	while (!list_empty(&buffer->mem_blocks)) {
		mem_block = list_first_entry(&buffer->mem_blocks,
                                            spy_mem_block_t, list);
		list_move(&mem_block->list, &server.free_mem_blocks);

		server.mem_blocks_used--;
	}

	buffer->read_pos    = 0;
	buffer->read_base   = 0;
	buffer->write_pos   = 0;
	buffer->write_base  = 0;
	buffer->cap         = 0;

	buffer->read_block  = NULL;
	buffer->write_block = NULL;
}

void spy_rw_buffer_reset_read(spy_rw_buffer_t *buffer, size_t pos)
{
	spy_mem_block_t *mem_block;

	assert(pos <= buffer->write_pos);

	buffer->read_block = NULL;
	buffer->read_base  = 0;
	buffer->read_pos   = 0;
	
	if (list_empty(&buffer->mem_blocks)) {
		assert(pos == 0);
		return;
	}

	mem_block = list_first_entry(&buffer->mem_blocks, spy_mem_block_t, list);
	
	while (buffer->read_base + mem_block->size < pos) {
		buffer->read_base += mem_block->size;
		
		mem_block = list_next_entry(mem_block, list);
	}
	
	buffer->read_block = mem_block;
	buffer->read_pos   = pos;
}

/*
 * get the next readable area.
 * return -1 mean no data for read, return 0 mean there is data for read.
 */
int spy_rw_buffer_next_readable(spy_rw_buffer_t *buffer, char **buf, size_t *size)
{
	spy_mem_block_t *mem_block;

	// no data for read
	if (buffer->read_pos == buffer->write_pos)
		return -1;
	
	assert(buffer->read_block);
	
	// jump to next mem_block
	if (buffer->read_base + buffer->read_block->size 
		== buffer->read_pos) {
        
		mem_block = list_next_entry(buffer->read_block, list);
		// assert ( (void*)mem_block != (void*)&buffer->mem_blocks );
		
		buffer->read_base  += buffer->read_block->size;
		buffer->read_block = mem_block;
	}
	
	*buf = buffer->read_block->buf + buffer->read_pos - buffer->read_base;
	*size = MIN(buffer->read_base + buffer->read_block->size - 
				buffer->read_pos, buffer->write_pos - buffer->read_pos);
	
	return 0;
}

/*
 * get the next writeable area.
 * return -1 mean no place for write, return 0 mean there is place for write
 */
int spy_rw_buffer_next_writeable(spy_rw_buffer_t *buffer, char **buf, size_t *size)
{
	spy_mem_block_t *mem_block;

	// no palce for write
	if (buffer->write_pos == buffer->cap)
		return -1;
	
	// jump to next mem_block
	if (buffer->write_base + buffer->write_block->size 
		== buffer->write_pos) {
        
		mem_block = list_next_entry(buffer->write_block, list);
		
		buffer->write_base      += buffer->write_block->size;
		buffer->write_block     = mem_block;
	}
        
	*buf = buffer->write_block->buf + buffer->write_pos - buffer->write_base;
	*size = buffer->write_base + buffer->write_block->size - buffer->write_pos;
	
	return 0;
}

/* return bytes read */
size_t spy_rw_buffer_read_n(spy_rw_buffer_t *buffer, char *buf, size_t size)
{
	spy_mem_block_t *mem_block;

	size_t data_len = buffer->write_pos - buffer->read_pos;
	size_t left     = size;
	size_t readable = 0, nread = 0;

	while (left > 0 && data_len > 0) {
		// jump to next mem block
		if (buffer->read_base + buffer->read_block->size == buffer->read_pos) {
			mem_block = list_next_entry(buffer->read_block, list);

			buffer->read_base  += buffer->read_block->size;
			buffer->read_block  = mem_block;
		}

		// current block readable size
		readable = MIN(buffer->write_pos, buffer->read_base + buffer->read_block->size) 
			- buffer->read_pos;

		nread = MIN(readable, left);
		memcpy((void*)(buf + size - left),
			   (void*)(buffer->read_block->buf + buffer->read_pos - buffer->read_base),
			   nread);

		buffer->read_pos += nread;
		data_len         -= nread;
		left             -= nread;
	}

	return size - left;
}

/* return bytes write */
size_t spy_rw_buffer_write_n(spy_rw_buffer_t *buffer, char *buf, size_t size)
{
	char *writeable;
	size_t len, nwrite, left = size;

	while (left > 0) {
		if (spy_rw_buffer_next_writeable(buffer, &writeable, &len) != 0)
			break;

		nwrite = MIN(len, left);
		memcpy((void*)writeable, (void*)(buf + size - left), nwrite);

		buffer->write_pos += nwrite;
		left              -= nwrite;
	}

	return size - left;
}
