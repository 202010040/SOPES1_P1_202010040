FROM ubuntu:20.04

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y \
    python3 \
    && rm -rf /var/lib/apt/lists/*

COPY ram_stress.py /app/ram_stress.py
RUN chmod +x /app/ram_stress.py

CMD ["python3", "/app/ram_stress.py"]
