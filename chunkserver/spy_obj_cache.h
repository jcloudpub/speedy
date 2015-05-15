#ifndef __SPY_OBJ_CACHE_H_
#define __SPY_OBJ_CACHE_H_

#include "spy_store.h"

void spy_init_io_jobs(size_t max_io_jobs);

void spy_free_io_job(spy_io_job_t *io_job);

spy_io_job_t* spy_gen_io_job();

#endif
