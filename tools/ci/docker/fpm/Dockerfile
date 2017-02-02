FROM ruby:2.3.3
RUN apt-get update && apt-get install -y rpm
RUN gem install --no-ri --no-rdoc fpm
COPY ./build.sh /build.sh
CMD ["/build.sh"]

