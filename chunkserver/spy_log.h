#ifndef __SPY_LOG_H__
#define __SPY_LOG_H__

#define DEBUG       0
#define INFO        1
#define WARN        2
#define ERROR       3

void spy_log(int level, char *fmt, ...);
#endif
