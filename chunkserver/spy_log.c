#include <unistd.h>
#include <stdlib.h>
#include <stdarg.h>
#include <assert.h>
#include <stdio.h>
#include <time.h>
#include <string.h>

#include "spy_log.h"
#include "spy_server.h"

void spy_log(int level, char *fmt, ...)
{
	int msg_len, time_len;
	time_t now;
	va_list ap;
	char buf[128], msg[1024];

	if (config.log_level > level) {
		return;
	}

	now = time(NULL);
	time_len = strftime(buf, 128, "[%Y-%m-%d %T] ", localtime(&now));
	assert(time_len < 128);

	memcpy((void*)msg, (void*)buf, time_len);

	va_start(ap,fmt);

	msg_len = vsnprintf(msg + time_len, 1023 - time_len, fmt, ap);
	assert(msg_len + time_len < 1024);

	va_end(ap);

	msg[msg_len + time_len] = '\n';

	write(server.log_fd, msg, msg_len + time_len + 1);
}
