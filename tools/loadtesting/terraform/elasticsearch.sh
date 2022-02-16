#!/bin/bash
yum update -y
yum install -y python3-pip git
pip3 install ansible

export TOKEN=`curl -X PUT "http://169.254.169.254/latest/api/token" -H "X-aws-ec2-metadata-token-ttl-seconds: 21600"`
export REPO=`curl -H "X-aws-ec2-metadata-token: $TOKEN" http://169.254.169.254/latest/meta-data/tags/instance/ansible_repository`
export PLAYBOOK_PATH=`curl -H "X-aws-ec2-metadata-token: $TOKEN" http://169.254.169.254/latest/meta-data/tags/instance/ansible_playbook_path`
export PLAYBOOK_FILE=`curl -H "X-aws-ec2-metadata-token: $TOKEN" http://169.254.169.254/latest/meta-data/tags/instance/ansible_playbook_file`
export BRANCH=`curl -H "X-aws-ec2-metadata-token: $TOKEN" http://169.254.169.254/latest/meta-data/tags/instance/ansible_branch`
export AWS_REGION=`curl -H "X-aws-ec2-metadata-token: $TOKEN" http://169.254.169.254/latest/meta-data/placement/region`

git clone "${REPO}" ansible
cd ansible
git checkout "${BRANCH}"
cd "${PLAYBOOK_PATH}"
ansible-playbook -c local "${PLAYBOOK_FILE}"

yum install -y docker
systemctl start docker.service
systemctl enable docker.service
curl -L "https://github.com/docker/compose/releases/download/1.29.2/docker-compose-$(uname -s)-$(uname -m)" -o /usr/bin/docker-compose
chmod +x /usr/bin/docker-compose
cat << EOT >docker-compose.yml
version: '2.2'
services:
  apm-server:
    image: docker.elastic.co/apm/apm-server:7.15.2
    depends_on:
      elasticsearch:
        condition: service_healthy
      kibana:
        condition: service_healthy
    cap_add: ["CHOWN", "DAC_OVERRIDE", "SETGID", "SETUID"]
    cap_drop: ["ALL"]
    ports:
    - 8200:8200
    networks:
    - elastic
    command: >
       apm-server -e
         -E apm-server.rum.enabled=true
         -E setup.kibana.host=kibana:5601
         -E setup.template.settings.index.number_of_replicas=0
         -E apm-server.kibana.enabled=true
         -E apm-server.kibana.host=kibana:5601
         -E output.elasticsearch.hosts=["elasticsearch:9200"]
    healthcheck:
      interval: 10s
      retries: 12
      test: curl --write-out 'HTTP %{http_code}' --fail --silent --output /dev/null http://localhost:8200/

  elasticsearch:
    image: docker.elastic.co/elasticsearch/elasticsearch:7.15.2
    environment:
    - bootstrap.memory_lock=true
    - cluster.name=docker-cluster
    - cluster.routing.allocation.disk.threshold_enabled=false
    - discovery.type=single-node
    - ES_JAVA_OPTS=-XX:UseAVX=2 -Xms1g -Xmx1g
    ulimits:
      memlock:
        hard: -1
        soft: -1
    volumes:
    - esdata:/usr/share/elasticsearch/data
    ports:
    - 9200:9200
    networks:
    - elastic
    healthcheck:
      interval: 20s
      retries: 10
      test: curl -s http://localhost:9200/_cluster/health | grep -vq '"status":"red"'

  kibana:
    image: docker.elastic.co/kibana/kibana:7.15.2
    depends_on:
      elasticsearch:
        condition: service_healthy
    environment:
      ELASTICSEARCH_URL: http://elasticsearch:9200
      ELASTICSEARCH_HOSTS: http://elasticsearch:9200
    ports:
    - 5601:5601
    networks:
    - elastic
    healthcheck:
      interval: 10s
      retries: 20
      test: curl --write-out 'HTTP %{http_code}' --fail --silent --output /dev/null http://localhost:5601/api/status

volumes:
  esdata:
    driver: local

networks:
  elastic:
    driver: bridge
EOT

docker-compose up -d
