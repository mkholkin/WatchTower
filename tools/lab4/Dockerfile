ARG BASE=watchtower-lab4-base:local
FROM ${BASE}
RUN apt-get update && apt-get install -y --no-install-recommends python3-venv && \
    python3 -m venv /opt/report && /opt/report/bin/pip install --no-cache-dir matplotlib==3.10.7 && \
    rm -rf /var/lib/apt/lists/*
ENV PATH="/opt/report/bin:${PATH}"
ENTRYPOINT ["python3"]
