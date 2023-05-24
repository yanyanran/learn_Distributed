#include <linux/watchdog.h>
#define O_RDWR 02
#define WDT_DEVICE_FILE "/dev/watchdog"

int main(void)
{
    int ret;
    int g_watchdog_fd = -1;
    int timeout = 1;
    int timeout_reset = 120;

    int sleep_time = 10;
    int feed_watchdog_time = 10;

    // 开启watchdog
    g_watchdog_fd = open(WDT_DEVICE_FILE, O_RDWR);
    if (g_watchdog_fd == -1)
    {
        printf("Error in file open WDT device file(%s)...\n", WDT_DEVICE_FILE);

        return 0;
    }
    // 获取watchdog的超时时间（heartbeat）
    ioctl(g_watchdog_fd, WDIOC_GETTIMEOUT, &timeout);
    printf("default timeout %d sec.\n", timeout);
    // 设置watchdog的超时时间（heartbeat）
    ioctl(g_watchdog_fd, WDIOC_SETTIMEOUT, &timeout_reset);
    printf("We reset timeout as %d sec.\n", timeout_reset);
    // 喂狗
    while (1)
    {
        // 喂狗
        ret = ioctl(g_watchdog_fd, WDIOC_KEEPALIVE, 0);
        // 喂狗也通过写文件的方式，向/dev/watchdog写入字符或者数字等
        //  static unsigned char food = 0;
        // write(g_watchdog_fd, &food, 1);
        if (ret != 0)
        {
            printf("Feed watchdog failed. \n");
            close(g_watchdog_fd);
            return -1;
        }
        else
        {
            printf("Feed watchdog every %d seconds.\n", sleep_time);
        }
        // feed_watchdog_time是喂狗的时间间隔，要小于watchdog的超时时间
        sleep(feed_watchdog_time);
    }
    // 关闭watchdog
    write(g_watchdog_fd, "V", 1);
    // 以下方式实测并不能关闭watchdog
    // ioctl(g_watchdog_fd, WDIOC_SETOPTIONS, WDIOS_DISABLECARD)
    close(g_watchdog_fd);
}