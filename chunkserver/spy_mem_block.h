#ifndef __SPY_MEM_BLOCK_H
#define __SPY_MEM_BLOCK_H

#include <stdint.h>
#include <stdlib.h>

#include "spy_list.h"

// for simple, every memory block has the same size
#define MEM_BLOCK_SIZE  (1 << 20)		// 1M
#define PREALLOC_COUNT  16
#define DEF_MEM_BLOCKS_LIMIT 1024	// 1G
#define MAX_MEM_BLOCKS_LIMIT (DEF_MEM_BLOCKS_LIMIT * 5)

#define MIN(a, b) ((a) < (b) ? (a) : (b))
#define MAX(a, b) ((a) > (b) ? (a) : (b))

typedef struct {
	struct list_head        list;
	uint64_t                size;
	char                    buf[];
} spy_mem_block_t;

void init_mem_blocks(struct list_head *mem_list, int prealloc_count);

#endif // __SPY_MEM_BLOCK_H
