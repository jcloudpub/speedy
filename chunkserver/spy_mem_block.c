#include "spy_mem_block.h"
#include "spy_log.h"

void init_mem_blocks(struct list_head *mem_list, int prealloc_count)
{
	char *ptr;
	int i;
	spy_mem_block_t *mem_block;

	INIT_LIST_HEAD(mem_list);

	if (prealloc_count > 0) {
		ptr = (char*)calloc((size_t)MEM_BLOCK_SIZE, (size_t)prealloc_count);
		if (!ptr) {
			spy_log(ERROR, "prealloc memory blocks failed!");
			exit(1);
		}

		i = prealloc_count;
		while (i-- > 0) {
			mem_block       = (spy_mem_block_t*)ptr;
			mem_block->size = MEM_BLOCK_SIZE - sizeof(spy_mem_block_t);

			list_add(&mem_block->list, mem_list);

			ptr += MEM_BLOCK_SIZE;
		}
	}
}
