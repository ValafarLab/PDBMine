FROM python:3.8-slim-buster

WORKDIR /pdbmine

RUN apt-get update && apt-get install -y dssp rsync

RUN echo "America/New_York" > /etc/timezone
RUN dpkg-reconfigure -f noninteractive tzdata

COPY requirements.txt requirements.txt
RUN pip3 install -r requirements.txt

COPY *.py .
COPY docker_config.yaml config.yaml

CMD [ "python3", "buildDB.py", "-y", "config.yaml", "-l", "INFO"]