FROM ruby:2.3.3
RUN apt-get update && apt-get install -y rpm debsigs
RUN gem install --no-ri --no-rdoc fpm
COPY ./build.sh /build.sh
COPY ./rpmmacros /root/.rpmmacros
CMD ["/build.sh"]

