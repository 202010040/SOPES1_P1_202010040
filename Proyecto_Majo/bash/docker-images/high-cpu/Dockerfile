FROM ubuntu:20.04

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y \
    stress-ng \
    bc \
    && rm -rf /var/lib/apt/lists/*

COPY cpu_stress.sh /app/cpu_stress.sh
RUN chmod +x /app/cpu_stress.sh

CMD ["/app/cpu_stress.sh"]
