FROM alpine:3.18
RUN set -ex \
      &&  apk --no-cache add build-base curl lua5.3 lua5.3-dev lua-cjson ca-certificates\
      && ln -s /usr/lib/lua5.3/liblua.a /usr/lib/liblua5.3.a \
      && ln -s /usr/lib/lua/5.3/cjson.so /usr/lib/cjson.so \
      && rm -rf /var/cache/apk/* 
