#ifndef _WATCH_DOG_H_
#define _WATCH_DOG_H_

int openHWDog(void);

void feedHWDog(int32_t fd);

int closeHWDog(int32_t fd);

#endif
