#include <errno.h>
#include <linux/watchdog.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <fcntl.h>
#include <string.h>
#include <sys/ioctl.h>

#include "watch_dog.h"

#define DEV_HWDOG "/dev/watchdog"

int openHWDog(void)
{
    int32_t fd, ret = -1;
    int32_t flags = WDIOS_ENABLECARD;
    fd = open(DEV_HWDOG, O_RDWR);
    if (fd < 0)
    {
        printf("can't open %s: %s\n", DEV_HWDOG, strerror(errno));
    }

    if ((ret = ioctl(fd, WDIOC_SETOPTIONS, &flags)) < 0)
    {
        printf("enable hardware dog failed: %d:%d,%s\n", ret, errno, strerror(errno));
        return -1;
    }
    /*
        if(ioctl(fd,WDIOC_GETSTATUS,&ret) < 0)
        {
            LOGERROR("hardware dog WDIOC_GETSTATUS failed!\n");
            return -1;
        }
        LOGINFO("WDIOC_GETSTATUS :%d\n",ret);
        */
    return fd;
}

inline void feedHWDog(int32_t fd)
{
    int dummy;
    if (ioctl(fd, WDIOC_KEEPALIVE, &dummy) < 0) {
        printf("feed hardware dog failed!\n");
        // LOGINFO("WDIOC_KEEPALIVE!\n");
    }
}

int closeHWDog(int32_t fd)
{
    int32_t flags = WDIOS_DISABLECARD;
    if (ioctl(fd, WDIOC_SETOPTIONS, &flags) < 0)
    {
        printf("close hardware dog failed: %s\n", strerror(errno));
        return -1;
    }

    close(fd);
    return 0;
}

int main()
{
    int32_t s32HWdogFd = 0;

    if ((s32HWdogFd = openHWDog()) < 0)
    {
        printf("openHWDog() Failed!!!\n");
        return -1;
    }

    while (1)
    {
        feedHWDog(s32HWdogFd);
        printf("main loop...\n");
        sleep(1);
    }
    closeHWDog(s32HWdogFd);
    return 0;
}