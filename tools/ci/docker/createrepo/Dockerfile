FROM centos
RUN yum update -y && yum install -y createrepo
COPY update.sh /update.sh
CMD ["/update.sh"]
