FROM ubuntu:18.04

RUN apt-get update && apt-get install -y openssh-server sudo
RUN mkdir /var/run/sshd

RUN useradd -ms /bin/bash -G sudo testuser
RUN echo 'testuser:testpass' | chpasswd

RUN useradd -ms /bin/bash -G sudo testuserkey
RUN echo 'testuserkey:testpass' | chpasswd

COPY test_rsa.pub /home/testuserkey/test_rsa.pub
RUN mkdir /home/testuserkey/.ssh
RUN chown testuserkey:testuserkey /home/testuserkey/.ssh
RUN cat /home/testuserkey/test_rsa.pub > /home/testuserkey/.ssh/authorized_keys
RUN chown testuserkey:testuserkey /home/testuserkey/.ssh/authorized_keys
RUN chmod 0600 /home/testuserkey/.ssh/authorized_keys
RUN echo "testuserkey ALL=(ALL) NOPASSWD: ALL" > /etc/sudoers.d/testuserkey
RUN chmod 0600 /etc/sudoers.d/testuserkey

EXPOSE 22
CMD ["/usr/sbin/sshd", "-D"]
