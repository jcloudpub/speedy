#ifndef __SPY_ADLER32_H__
#define __SPY_ADLER32_H__

#include "spy_rw_buffer.h"

typedef unsigned char  Byte;  /* 8 bits */
typedef unsigned int   uInt;  /* 16 bits or more */
typedef unsigned long  uLong; /* 32 bits or more */

uLong spy_adler32(uLong adler, const Byte *buf, uInt len);

uint64_t spy_buffer_adler32(uint64_t adler, spy_rw_buffer_t *buffer, size_t len);

#endif
