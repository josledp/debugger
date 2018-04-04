FROM ubuntu:16.04
MAINTAINER José Luis Ledesma <joseluis.ledesma@gmail.com>
RUN apt-get update &&             \
    apt-get install -y            \
      strace                      \
      gdb                         \
      tcpdump                     \
      util-linux                  \
      apt-transport-https         \
      ca-certificates             \
      curl                        \
      software-properties-common  \
      &&                          \
    apt-get clean
RUN sh -c "curl -fsSL https://download.docker.com/linux/ubuntu/gpg | apt-key add -"
RUN add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu xenial stable"
RUN apt-get update &&   \
    apt-get install     \
      docker-ce         \
      &&                \
    apt-get clean
CMD ["sleep", "72000"]
