FROM ubuntu
RUN apt-key adv --keyserver keys.gnupg.net --recv-keys 9E3E53F19C7DE460 && \
    apt-get update -y && apt-get install -y aptly
COPY update.sh /update.sh
CMD ["/update.sh"]
