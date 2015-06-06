#ifndef __SPY_RW_STREAM_H
#define __SPY_RW_STREAM_H

#include <stdint.h>
#include <stdlib.h>

#include "spy_list.h"
#include "spy_log.h"
#include "spy_mem_block.h"

// linked read write buffer
typedef struct {
	struct list_head        mem_blocks;

	size_t                  read_pos;       // current read position
	size_t                  write_pos;      // current write postion
	size_t                  cap;            // capacity

	spy_mem_block_t         *read_block;    // current read block
	spy_mem_block_t         *write_block;   // current write block
	size_t                  read_base;      // current read block base pos
	size_t                  write_base;     // current write block base pos
} spy_rw_buffer_t;

// init spy_rw_buffer_t struct
void spy_rw_buffer_init(spy_rw_buffer_t *buffer);

// expand buffer's capacity
int spy_rw_buffer_expand(spy_rw_buffer_t *buffer);

// reset read && write
void spy_rw_buffer_reset(spy_rw_buffer_t *buffer);

// reset read to the specified position(for repeated read)
void spy_rw_buffer_reset_read(spy_rw_buffer_t *buffer, size_t pos);

// get the next readable area.
// return -1 mean no data for read, 0 mean there is data for read.
int spy_rw_buffer_next_readable(spy_rw_buffer_t *buffer,
        char **buf/*out*/, size_t *size/*out*/);

// get the next writeable area.
// return -1 mean no place for write, return 0 mean there is place for write
int spy_rw_buffer_next_writeable(spy_rw_buffer_t *buffer,
        char **buf/*out*/, size_t *size/*out*/);

// read n bytes from buffer, return bytes read
size_t spy_rw_buffer_read_n(spy_rw_buffer_t *buffer, char *buf, size_t size);

// write n bytes to buffer, return bytes write
size_t spy_rw_buffer_write_n(spy_rw_buffer_t *buffer, char *buf, size_t size);

#endif // __SPY_RW_STREAM_H
