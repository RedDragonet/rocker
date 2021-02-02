#define _GNU_SOURCE
#include <errno.h>
#include <fcntl.h>
#include <sched.h>
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>

enum sync_t {
        SYNC_SETNS_ACK = 0x40, /* 完成SETNS操作 */
};

static int pipe_fd(char *env) {
        char *pipe, *endptr;

        pipe = getenv(env);
        if (pipe == NULL || *pipe == '\0')
                return -1;

        int pipefd = strtol(pipe, &endptr, 10);
        if (*endptr != '\0')
                return -1;

        return pipefd;
}

void nsexec() {
        //获取 容器进程ID
        char *container_id = getenv("CONTAINER_PID");
        //环境变量为空，说明是不是通过 exec 启动
        if (!container_id) {
                return;
        }
        fprintf(stdout, "C: container_id %s\n", container_id);

        char nspath[1024];
        char *namespaces[] = {"ipc", "uts", "net", "pid", "mnt"};
        for (int i = 0; i < 5; i++) {
                sprintf(nspath, "/proc/%s/ns/%s", container_id, namespaces[i]);
                fprintf(stdout, "C: setns open %s\n", nspath);
                int fd = open(nspath, O_RDONLY);
                if (fd == -1) {
                        fprintf(stdout, "C: open namespaces faild %d ,errno = %d\n", fd, errno);
                        exit(-1);
                }

                if (setns(fd, 0) == -1) {
                        fprintf(stdout, "C: setns failed %d ,errno = %d\n", fd, errno);
                        exit(-1);
                }
                close(fd);
        }

        enum sync_t s;
        int parent_fd = pipe_fd("CONTAINER_PIPE_PARENT");
        close(parent_fd);
        if (write(parent_fd, &s, sizeof(s)) != sizeof(s)) {
                fprintf(stdout, "C: write parent_fd failed\n");
        }
        fprintf(stdout, "C: write parent_fd %d done\n", parent_fd);

        int command_fd = pipe_fd("CONTAINER_PIPE_COMMAND");
        //等待命令传输
        char command[1024];
        int len = read(command_fd, &command, sizeof(command));
        fprintf(stdout, "C: read command_fd %d %s\n", len, command);

        // 直接执行容器中的命令
        system(command);

        //安全退出
        exit(0);
}
