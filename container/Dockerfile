FROM ubuntu:16.04
MAINTAINER José Luis Ledesma <joseluis.ledesma@gmail.com>
RUN apt-get update &&  \
    apt-get install -y \
      strace           \
      gdb              \
      tcpdump          \
      &&  \
    apt-get clean
CMD ["/bin/bash"]
