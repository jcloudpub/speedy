#ifndef __SPY_WORK_H__
#define __SPY_WORK_H__

#define WORKER_FN

#include <pthread.h>
#include <stdlib.h>
#include <unistd.h>
#include <stdint.h>

#include "spy_list.h"
#include "spy_atomic.h"

#define DEF_WQ_NR_THRS 10

struct work;

typedef void (*spy_work_func_t)(struct work *work);

typedef struct work {
	spy_work_func_t           fn;
	spy_work_func_t           done;
	struct list_head          w_list;
} spy_work_t;

typedef struct {
	struct list_head          pending_list;
	struct list_head          finished_list;

	pthread_mutex_t           pending_lock;
	pthread_mutex_t           finished_lock;
	pthread_cond_t            pending_cond;

	spy_atomic_t              nr_threads;
	spy_atomic_t              nr_works;
	
	uint64_t                  last_expand_time;
	int                       finished_event_fd;
} spy_work_queue_t;

spy_work_queue_t *spy_create_work_queue(int nr_threads);
void spy_queue_work(spy_work_queue_t *work_queue, spy_work_t *work);
#endif
