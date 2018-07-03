#!/bin/bash

docker run --rm -it \
    -e XPACK_MONITORING_ENABLED=false \
    -p 12201:12201/udp \
    -v /Users/benkeil/development/go/src/github.com/benkeil/kubernetes-event-processor/docker/pipiline/:/usr/share/logstash/pipeline/ \
    docker.elastic.co/logstash/logstash:6.3.0