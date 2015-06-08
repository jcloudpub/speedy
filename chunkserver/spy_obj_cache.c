#include <assert.h>
#include <string.h>

#include "spy_obj_cache.h"
#include "spy_server.h"

void spy_init_io_jobs(size_t max_io_jobs)
{
	spy_io_job_t *io_job;

	assert (max_io_jobs > 0);

	io_job = (spy_io_job_t *)calloc(max_io_jobs, sizeof(spy_io_job_t));
	assert (io_job);

	while (max_io_jobs--) {
		list_add(&io_job->oc_list, &server.free_io_jobs);

		io_job++;
	}
}

spy_io_job_t* spy_gen_io_job()
{
	spy_io_job_t *io_job;

	if (list_empty(&server.free_io_jobs))
		return NULL;

	io_job = list_first_entry(&server.free_io_jobs, spy_io_job_t, oc_list);
	list_del(&io_job->oc_list);

	return io_job;
}

void spy_free_io_job(spy_io_job_t *io_job)
{
	memset((void*)io_job, 0, sizeof(spy_io_job_t));

	list_add(&io_job->oc_list, &server.free_io_jobs);
}
