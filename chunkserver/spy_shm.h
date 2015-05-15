#ifndef __SHM_H__
#define __SHM_H__
#include <sys/mman.h>
#include <asm/types.h>
#include <sys/types.h>
#include <sys/ioctl.h>
#include <linux/fs.h>
#include <assert.h>
#include <sys/shm.h>
#include <sys/ipc.h>
#include <errno.h>
#include <stdlib.h>
#include <signal.h>

#define REPORT_SHM_KEY 8701
#define REPORT_SHM_SIZE 4096

char* spy_create_and_attach_shm(key_t shm_key, size_t shm_size);
#endif
