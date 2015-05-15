#ifndef __SPY_UTILS_H__
#define __SPY_UTILS_H__

#include <errno.h>
#include <unistd.h>

#define VARIANT_HEAD (1 << 7)
#define byte unsigned char

void spy_make_daemonize();
uint64_t spy_current_time_sec();
uint64_t spy_current_time_usec();

void spy_mach_write_to_1(byte *b, uint8_t data);
uint8_t spy_mach_read_from_1(byte *b);
void spy_mach_write_to_2(byte *b, uint16_t data);
uint16_t spy_mach_read_from_2(byte *b);
void spy_mach_write_to_4(byte *b, uint32_t data);
uint32_t spy_mach_read_from_4(byte *b);
void spy_mach_write_to_8(byte *b, uint64_t data);
uint64_t spy_mach_read_from_8(byte *b);
int spy_mach_write_variant_4(byte *b, uint32_t data);
uint32_t spy_mach_read_variant_4(byte *b, uint32_t *value);

ssize_t spy_pwrite(int fd, void *buf, size_t count, off_t offset);
ssize_t spy_pread(int fd, void *buf, size_t count, off_t offset);

#endif
