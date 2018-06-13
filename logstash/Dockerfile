FROM docker.elastic.co/logstash/logstash:6.2.4
RUN rm -f /usr/share/logstash/pipeline/logstash.conf
RUN logstash-plugin install logstash-output-mongodb
ADD logstash/pipeline/ /usr/share/logstash/pipeline/
ADD logstash/config/ /usr/share/logstash/config/