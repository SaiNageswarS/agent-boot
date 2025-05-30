FROM ubuntu:latest

# Essential for using tls
RUN apt-get update
RUN apt-get install ca-certificates -y
RUN update-ca-certificates

# Install mupdf
RUN apt-get install -y mupdf mupdf-tools

# web port
EXPOSE 8081
# grpc port
EXPOSE 50051

ADD build/search-core /app/search-core
ADD config.ini /app/config.ini
RUN ls -l

CMD /app/search-core