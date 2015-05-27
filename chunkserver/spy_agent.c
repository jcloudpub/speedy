#include <sys/time.h>
#include <pthread.h>
#include <assert.h>
#include <sys/socket.h>
#include <sys/types.h>
#include <netinet/in.h>
#include <netdb.h>
#include <stdio.h>
#include <string.h>
#include <stdlib.h>
#include <unistd.h>
#include <errno.h>
#include <arpa/inet.h> 
#include <inttypes.h>

#include "spy_server.h"
#include "spy_agent.h"
#include "spy_log.h"

char *master_report_uri = "/v1/chunkserver/reportinfo";

char *http_text = "POST %s HTTP/1.1\r\n"
	"Host: %s:%d\r\n"
	"Content-Type: text/json\r\n"
	"Content-Length: %d\r\n\r\n"
	"%s";

char *json_data = "{\"GroupId\": %d,"
	"\"Ip\":\"%s\","
	"\"Port\":%d,"
    "\"Status\": %d,"
	"\"TotalFreeSpace\":%"PRIu64","
	"\"MaxFreeSpace\":%"PRIu64","
    "\"PendingWrites\": %d,"
	"\"WritingCount\": %d,"
	"\"ReadingCount\": %d,"
    "\"DataDir\": \"%s\","
    "\"TotalChunks\":%d,"
     "\"ConnectionsCount\":%d}";

static int spy_write(int fd, char *buf, int count)
{
	int nwritten, totlen = 0;
	while (totlen != count) {
		nwritten = write(fd, buf, count - totlen);
		if (nwritten == 0) return totlen;
		if (nwritten == -1) return -1;
		totlen += nwritten;
		buf += nwritten;
	}

	return totlen;
}

int spy_connect_master()
{
	int fd;
	struct sockaddr_in addr; 

	if ((fd = socket(AF_INET, SOCK_STREAM, 0)) < 0) {
		spy_log(ERROR, "create socket failed %s", strerror(errno));
		return -1;
	}

	memset(&addr, 0, sizeof(addr));
	
	addr.sin_family = AF_INET;
	addr.sin_port   = htons(config.master_port);

	if (inet_pton(AF_INET, config.master_addr, &addr.sin_addr) <= 0) {
		spy_log(ERROR, "inet_pton error %s", strerror(errno));
		close(fd);

		return -1;
    }

	if (connect(fd, (struct sockaddr *)&addr, sizeof(addr)) < 0) {
		spy_log(ERROR, "connect master error %s", strerror(errno));
		close(fd);

		return -1;
	}

	return fd;
}

void spy_spinlock(spy_atomic_t *lock)
{
	int i, n, spin = 20;

	if (spy_atomic_cmp_and_set(lock, 0, 1)) {
		return;
	}

	for (;;) {
		for (n = 1; n < spin; n <<= 1) {
		
			for (i = 0; i < n; i++) {
				__asm__ ("pause");
			}
		
			if (lock->counter == 0 && spy_atomic_cmp_and_set(lock, 0, 1)) {
				return;
			}
		}
	
		sched_yield();
	}
}

void *spy_report_routine(void *arg)
{
	int fd, n, clen;
	char json_content[2048];
	char http_content[4096];
	char http_post_addr[256];

	char http_resp[4096];
	
	(void)arg;

	for (;;) {
		fd = spy_connect_master();
		
		if (fd < 0) {
			spy_log(ERROR, "failed to report info to master");
			sleep(1);
			continue;
		}

		spy_spinlock(&report_info.lock);
		
		n = snprintf(json_content, 2048, json_data, config.server_id,
					 config.bind_addr, config.port, report_info.status,
					 report_info.total_free_space,
					 report_info.max_free_space,
					 report_info.pending_writes,
					 report_info.writing_count,
					 report_info.reading_count,
					 config.data_dir,
					 report_info.n_chunks,
					 report_info.conn_count);

		assert(n < 2048);

		spy_atomic_set(&report_info.lock, 0);

		n = snprintf(http_post_addr, 256, "http://%s:%d%s", 
					 config.master_addr, config.master_port, 
					 master_report_uri);
		assert(n < 256);

		n = snprintf(http_content, 4096, http_text, http_post_addr,
					 config.bind_addr, config.port, strlen(json_content),
					 json_content);
		assert(n < 4096);

		clen = strlen(http_content);
		
		n = spy_write(fd, http_content, clen);
		
		if (n != clen) {
			spy_log(ERROR, "write report info error ret=%d", n);
			close(fd);
			continue;
		}

		n = read(fd, http_resp, 4096);

		if (n > 0) {
			http_resp[n] = '\0';

			if (strncmp(http_resp, "HTTP/1.1 200 OK", 15) != 0) {
				spy_log(ERROR, "report chunk server status failed, http resp %s", http_resp);
			}
		} else {
			spy_log(ERROR, "read report chunk server status resp failed, nread %d, err %s", 
					n, strerror(errno));
		}

		close(fd);

		sleep(DEF_REPORT_INTERVAL);
	}
}

void spy_start_agent_thread()
{
	pthread_t thread;

	spy_atomic_set(&report_info.lock, 0);

	assert(pthread_create(&thread, NULL, spy_report_routine, NULL) == 0);
}
