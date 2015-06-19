#include <fcntl.h>
#include <unistd.h>
#include <stdlib.h>
#include <stdarg.h>
#include <stdint.h>
#include <stdio.h>
#include <sys/time.h>
#include <assert.h>
#include <string.h>

#include "spy_utils.h"

ssize_t spy_pread(int fd, void *buf, size_t count, off_t offset)
{
	char *p = buf;	
	ssize_t total = 0;

	while (count > 0) {
		ssize_t readn = pread(fd, p, count, offset);
		if (readn < 0 && (errno == EAGAIN || errno == EINTR))
			continue;
		if (readn <= 0) {
			return -1;
		}

		count -= readn;
		p += readn;
		total += readn;
		offset += readn;
	}

	return total;
}

ssize_t spy_pwrite(int fd, void *buf, size_t count, off_t offset)
{
	char *p = buf;	
	ssize_t total = 0;

	while (count > 0) {
		ssize_t written = pwrite(fd, p, count, offset);
		if (written < 0 && (errno == EAGAIN || errno == EINTR))
			continue;
		if (written <= 0) {
			return -1;
		}

		count -= written;
		p += written;
		total += written;
		offset += written;
	}

	return total;
}

void spy_make_daemonize()
{
	int fd;

	if (fork() != 0) exit(0);
	setsid();

	if ((fd = open("/dev/null",O_RDWR,0)) != -1) {
		dup2(fd,STDIN_FILENO);
		dup2(fd,STDOUT_FILENO);		   
		dup2(fd,STDERR_FILENO);		   

		if (fd > STDERR_FILENO) close(fd);
	}
}

uint64_t spy_current_time_sec()
{
	struct timeval tv;

	gettimeofday(&tv, NULL);	

	return (uint64_t)tv.tv_sec;
}

uint64_t spy_current_time_usec()
{
	struct timeval tv;

	gettimeofday(&tv, NULL);	

	return (uint64_t)tv.tv_sec * 1000 * 1000 + tv.tv_usec;
}

void spy_mach_write_to_1(byte *b, uint8_t data)
{
	assert(b);
	b[0] = (byte)(data & 0xFFUL);
}

uint8_t spy_mach_read_from_1(byte *b)
{
	assert(b);
	return (uint8_t)(b[0]);
}
 
void spy_mach_write_to_2(byte *b, uint16_t data)
{
	assert(b);
	b[0] = (byte)(data >> 8);
	b[1] = (byte)(data);
}

uint16_t spy_mach_read_from_2(byte *b)
{
	assert(b);
	return ((uint16_t)(b[0] << 8) | (uint16_t)b[1]);
}

void spy_mach_write_to_4(byte *b, uint32_t data)
{
	assert(b);
	b[0] = (byte)(data >> 24);
	b[1] = (byte)(data >> 16);
	b[2] = (byte)(data >> 8);
	b[3] = (byte)(data);
}

uint32_t spy_mach_read_from_4(byte *b)
{
	assert(b);
	return ((uint32_t)(b[0] << 24) | 
			(uint32_t)(b[1] << 16) |
			(uint32_t)(b[2] << 8)  |
			(uint32_t)(b[3]));
}

void spy_mach_write_to_8(byte *b, uint64_t data)
{
	assert(b);
	b[0] = (byte)(data >> 56);
	b[1] = (byte)(data >> 48);
	b[2] = (byte)(data >> 40);
	b[3] = (byte)(data >> 32);
	b[4] = (byte)(data >> 24);
	b[5] = (byte)(data >> 16);
	b[6] = (byte)(data >> 8);
	b[7] = (byte)(data);
}

uint64_t spy_mach_read_from_8(byte *b)
{
	assert(b);
	return (((uint64_t)b[0] << 56) | 
			((uint64_t)b[1] << 48) |
			((uint64_t)b[2] << 40) |
			((uint64_t)b[3] << 32) |
			((uint64_t)b[4] << 24) |
			((uint64_t)b[5] << 16) |
			((uint64_t)b[6] << 8)  |
			 (uint64_t)b[7]);
}

int spy_mach_write_variant_4(byte *b, uint32_t data)
{
	if (data < (1 << 7)) { 
		b[0] = (byte)(data);
		return 1;
	} else if (data < (1 << 14)) {
		b[0] = (byte)(data) | VARIANT_HEAD;
		b[1] = (byte)(data >> 7);
		return 2;
	} else if (data < (1 << 21)) { 
		b[0] = (byte)(data) | VARIANT_HEAD;
		b[1] = (byte)(data >> 7) | VARIANT_HEAD;		
		b[2] = (byte)(data >> 14);
		return 3;
	} else if (data < (1 << 28)) { 
		b[0] = (byte)(data) | VARIANT_HEAD;
		b[1] = (byte)(data >> 7) | VARIANT_HEAD;		
		b[2] = (byte)(data >> 14) | VARIANT_HEAD;		
		b[3] = (byte)(data >> 21);
		return 4;
	} else { 
		b[0] = (byte)(data) | VARIANT_HEAD;
		b[1] = (byte)(data >> 7) | VARIANT_HEAD;
		b[2] = (byte)(data >> 14) | VARIANT_HEAD;		
		b[3] = (byte)(data >> 21) | VARIANT_HEAD;
		b[4] = (byte)(data >> 28);
		return 5;
	}
}

uint32_t spy_mach_read_variant_4(byte *b, uint32_t *value)
{
	uint32_t res = 0,shift;
	byte *p = b;
	for (shift = 0; shift <= 28; shift += 7) {
		if (!(*p & VARIANT_HEAD)) {
			//enough,the last one
			res |= (*p << shift);
			*value = res;
			break;
		} else {
			res |= ((*p & (VARIANT_HEAD - 1)) << shift);		
		}
		p++;
	}

	return p - b + 1;
}

int spy_string_ends_with(const char *str, const char *suffix) 
{
	size_t lstr, lsuffix;

	if (!str || !suffix) {
		return 0;
	}

	lstr    = strlen(str);
	lsuffix = strlen(suffix);

	if (lsuffix > lstr) {
		return 0;
	}

	return strncmp(str + lstr - lsuffix, suffix, lsuffix) == 0;
}
