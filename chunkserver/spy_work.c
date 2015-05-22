#include <assert.h>
#include <errno.h>
#include <unistd.h>
#include <sys/eventfd.h>
#include <stddef.h>
#include <stdio.h>
#include <stdint.h>
#include <sys/types.h>

#include "spy_utils.h"
#include "spy_list.h"
#include "spy_work.h"

#define SHRINK_PERIOD_PROTECTION 10

static int spy_wq_need_shrink(spy_work_queue_t *wq)
{
	int n;
	uint64_t current_time   = spy_current_time_sec();
	size_t nr_works         = spy_atomic_read(&wq->nr_works);
	size_t nr_threads       = spy_atomic_read(&wq->nr_threads);

	nr_works        += 1;
	nr_threads      += 1;

	if (nr_works < nr_threads / 2 && nr_threads > DEF_WQ_NR_THRS / 2
		&& current_time > wq->last_expand_time + SHRINK_PERIOD_PROTECTION) {

		return 1;
	}

	return 0;
}

static int spy_wq_need_expand(spy_work_queue_t *wq)
{
	size_t nr_threads = spy_atomic_read(&wq->nr_threads);
	size_t nr_works = spy_atomic_read(&wq->nr_works);

	nr_threads      += 1;
	nr_works        += 1;

	if (nr_threads < nr_works / 2 && nr_threads < 2 * DEF_WQ_NR_THRS)
		return 1;
	
	return 0;
}

static void spy_finish_work_notify(int event_fd)
{
	int ret;
	uint64_t dummy = 1;

	do {
		ret = write(event_fd, &dummy, 8);
	} while (ret < 0 && (errno == EAGAIN || errno == EINTR));

	assert(ret >= 0);

}

static void *spy_worker_routine(void *arg)
{
	spy_work_t *work;
	spy_work_queue_t *wq = arg;

	while (1) {
		pthread_mutex_lock(&wq->pending_lock);

		if (spy_wq_need_shrink(wq)) {
			spy_atomic_sub(1, &wq->nr_threads);
			pthread_mutex_unlock(&wq->pending_lock);

			break;
		}
		
		while (list_empty(&wq->pending_list)) {
			pthread_cond_wait(&wq->pending_cond, &wq->pending_lock);
		}

		work = list_first_entry(&wq->pending_list, spy_work_t, w_list);
		list_del(&work->w_list);

		pthread_mutex_unlock(&wq->pending_lock);

		if (work->fn) 
			work->fn(work);

		pthread_mutex_lock(&wq->finished_lock);
		list_add_tail(&work->w_list, &wq->finished_list);
		pthread_mutex_unlock(&wq->finished_lock);

		spy_atomic_sub(1, &wq->nr_works);
		spy_finish_work_notify(wq->finished_event_fd);
	}
}

static int spy_create_worker_threads(spy_work_queue_t *work_queue, int nr_threads)
{
	int ret;
	pthread_t thread;

	while (spy_atomic_read(&work_queue->nr_threads) < nr_threads) {
		ret = pthread_create(&thread, NULL, spy_worker_routine, work_queue);
		if (ret) 
			return -1;

		spy_atomic_add(1, &work_queue->nr_threads);
	}

	return 0;
}

spy_work_queue_t *spy_create_work_queue(int nr_threads)
{
	int ret, n;

	spy_work_queue_t *work_queue = malloc(sizeof(spy_work_queue_t));
	assert(work_queue);

	INIT_LIST_HEAD(&work_queue->pending_list);
	INIT_LIST_HEAD(&work_queue->finished_list);

	n = pthread_mutex_init(&work_queue->pending_lock, NULL);
	assert(n == 0);

	n = pthread_mutex_init(&work_queue->finished_lock, NULL);
	assert(n == 0);

	n = pthread_cond_init(&work_queue->pending_cond, NULL);
	assert(n == 0);

	spy_atomic_set(&work_queue->nr_works, 0);
	spy_atomic_set(&work_queue->nr_threads, 0);
	work_queue->last_expand_time = spy_current_time_sec();

	work_queue->finished_event_fd = eventfd(0, EFD_NONBLOCK);
	assert(work_queue->finished_event_fd);

	ret = spy_create_worker_threads(work_queue, nr_threads);
	assert(ret == 0);

	return work_queue;
}

void spy_queue_work(spy_work_queue_t *wq, spy_work_t *work)
{
	spy_atomic_add(1, &wq->nr_works);

	INIT_LIST_HEAD(&work->w_list);

	pthread_mutex_lock(&wq->pending_lock);

	if (spy_wq_need_expand(wq)) {
		spy_create_worker_threads(wq, spy_atomic_read(&wq->nr_threads) * 2);
		wq->last_expand_time = spy_current_time_sec();
	}

	list_add_tail(&work->w_list, &wq->pending_list);
	pthread_mutex_unlock(&wq->pending_lock);

	pthread_cond_signal(&wq->pending_cond);
}
