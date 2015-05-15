#ifndef __SPY_AGENT_H__
#define __SPY_AGENT_H__

#define DEF_REPORT_INTERVAL 10

#include "spy_atomic.h"

typedef struct {
	spy_atomic_t           lock;

	int                    pending_writes;
	int                    writing_count;
	int                    reading_count;
	int                    n_chunks;
	int                    conn_count;
	int                    status;	

	uint64_t               total_free_space;
	uint64_t               max_free_space;
} spy_report_info_t;

void spy_start_agent_thread();
#endif
