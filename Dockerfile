FROM ubuntu:latest

# Essential for using tls
RUN apt-get update
RUN apt-get install ca-certificates -y
RUN update-ca-certificates

# web port
EXPOSE 8081
# grpc port
EXPOSE 50051

ADD build/agent-boot /app/agent-boot
ADD config.ini /app/config.ini
RUN ls -l

CMD /app/agent-boot