FROM denkhaus/alpine-golang:3.3
MAINTAINER denkhaus <denkhaus@github>

RUN apk add --update bash curl && \
  	rm -rf /var/cache/apk/*

# get splunk-pump
RUN go get -u github.com/denkhaus/splunk-pump

ENTRYPOINT ["splunk-pump"]