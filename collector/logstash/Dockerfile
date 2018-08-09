FROM docker.elastic.co/logstash/logstash:6.2.4

# Leave the pipelines to be configured via
# docker stack configs or bind mounts
RUN rm -f /usr/share/logstash/pipeline/logstash.conf

# Install the MongoDB plugin
RUN logstash-plugin install logstash-output-mongodb

# Install the latest netflow codec
RUN logstash-plugin update logstash-codec-netflow

# Add a directory to cache ipfix/ netflow templates in
# To maintain the cache across invocations it must be volumed
RUN mkdir /usr/share/logstash/template_cache

# Add the IPFIX/ Netflow Decoder pipeline
# NOTE: The MongoDB output pipeline should be added
# via bind mount or via docker configs
ADD pipeline/ /usr/share/logstash/pipeline/
