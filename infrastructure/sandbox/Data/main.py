'''Ships alb logs from s3 to es'''
import os
import re
import gzip
import logging
import hashlib
import urllib.parse
import boto3
from elasticsearch import Elasticsearch
from elasticsearch.helpers import bulk

print('Loading function')

s3 = boto3.client('s3')

logging.basicConfig(level=logging.INFO)

def parse_int(thing):
    try:
        return int(thing)
    except:
        return None

fields = (
    ("type", str),
    ("time", str),
    ("elb", str),
    ("client_ip", str),
    ("client_port", parse_int),
    ("target_ip", str),
    ("target_port", parse_int),
    ("request_processing_time", float),
    ("target_processing_time", float),
    ("response_processing_time", float),
    ("elb_status_code", parse_int),
    ("target_status_code", str),
    ("received_bytes", parse_int),
    ("sent_bytes", parse_int),
    ("request_verb", str),
    ("request_url", lambda a: urllib.parse.urlsplit(a)._asdict()),
    ("request_proto", str),
    ("user_agent", str),
    ("ssl_cipher", str),
    ("ssl_protocol", str),
    ("target_group_arn", str),
    ("trace_id", str),
    ("domain_name", str),
    ("chosen_cert_arn", str),
    ("matched_rule_priority", str),
    ("request_creation_time", str),
    ("actions_executed", str),
    ("redirect_url", str),
    ("lambda_error_reason", str),
    ("target_port_list", str),
    ("target_status_code_list", str),
    ("classification", str),
    ("classification_reason", str),
)

regex = r'([^ ]*) ([^ ]*) ([^ ]*) ([^ ]*):([0-9]*) ([^ ]*)[:-]([0-9]*) ([-.0-9]*) ([-.0-9]*) ([-.0-9]*) (|[-0-9]*) (-|[-0-9]*) ([-0-9]*) ([-0-9]*) \"([^ ]*) (.*) (- |[^ ]*)\" \"([^\"]*)\" ([A-Z0-9-_]+) ([A-Za-z0-9.-]*) ([^ ]*) \"([^\"]*)\" \"([^\"]*)\" \"([^\"]*)\" ([-.0-9]*) ([^ ]*) \"([^\"]*)\" \"([^\"]*)\" \"([^ ]*)\" \"([^\s]+?)\" \"([^\s]+)\" \"([^ ]*)\" \"([^ ]*)\"'
matcher = re.compile(regex)

def do_file(bucket, key):
    '''Generates log lines'''
    search = Elasticsearch([os.environ['ES_URL']])
    out = []
    response = s3.get_object(Bucket=bucket, Key=key)
    with gzip.GzipFile(fileobj=response["Body"]) as handle:
        for line in handle:
            match = matcher.match(line)
            if not match:
                raise line
            thing = {i[0]: i[1](match.group(n+1)) for n, i in enumerate(fields)}
            thing['_index'] = 'sandbox-prod'
            thing['_id'] = hashlib.sha256(line.encode('utf8')).hexdigest()
            if thing['elb_status_code'] == 200:
                out.append(thing)
    bulk(search, out, doc_type='doc', chunk_size=100)


def lambda_handler(event, context):
    #print("Received event: " + json.dumps(event, indent=2))

    # Get the object from the event and show its content type
    bucket = event['Records'][0]['s3']['bucket']['name']
    key = urllib.parse.unquote_plus(event['Records'][0]['s3']['object']['key'], encoding='utf-8')
    do_file(bucket, key)
