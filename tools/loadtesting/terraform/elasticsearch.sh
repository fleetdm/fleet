#!/bin/bash
yum update -y
yum install -y python3-pip git
pip3 install ansible boto3

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
chown -R ansible:ansible ~ansible/ansible
rm -rf ansible
