#include <stdio.h>
#include <unistd.h>

int main(void)
{
    while (1)
    {
        // 模拟长时间运行的任务
        printf("Watchdog demo - Task is running...\n");
        sleep(10); // 每10s执行一次
    }

    return 0;
}
