#define _GNU_SOURCE
#include <unistd.h>
#include <stdio.h>
#include <stdlib.h>
#include <fcntl.h>
#include <sched.h>

void nsexec() {
    //获取 容器进程ID
    char *container_id = getenv("CONTAINER_PID");
    //环境变量为空，说明是不是通过 exec 启动
    if(!container_id){
        return ;
    }
    char *container_cmd = getenv("CONTAINER_CMD");

    char nspath[1024];
    char *namespaces[] = { "ipc", "uts", "net", "pid", "mnt" };
    for (int i=0; i<5; i++) {
        sprintf(nspath, "/proc/%s/ns/%s", container_id, namespaces[i]);
        int fd = open(nspath, O_RDONLY);

        if (setns(fd, 0) == -1) {
            exit(-1);
        }
        close(fd);
    }
    system(container_cmd);
    //安全退出
    exit(0);
}