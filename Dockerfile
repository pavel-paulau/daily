FROM alpine:3.5

MAINTAINER Pavel Paulau <pavel@couchbase.com>

EXPOSE 8008

ENV CB_HOST ""
ENV CB_PASS ""

COPY app app
COPY daily /usr/local/bin/daily

CMD ["daily"]
