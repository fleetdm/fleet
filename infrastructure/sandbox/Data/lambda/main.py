'''Ships alb logs from s3 to es'''
import os
import re
import gzip
import logging
import hashlib
import geohash
import urllib.parse
import boto3
from geoip import geolite2
from elasticsearch import Elasticsearch
from elasticsearch.helpers import bulk
from elasticsearch.serializer import JSONSerializer

print('Loading function')

s3 = boto3.client('s3')

logger = logging.getLogger()
logger.setLevel(logging.DEBUG)
es_logger = logging.getLogger('elasticsearch')
es_logger.setLevel(logging.DEBUG)

class SetEncoder(JSONSerializer):
    def default(self, obj):
        if isinstance(obj, frozenset):
            return list(obj)
        return JSONSerializer.default(self, obj)

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

REGEX = r'([^ ]*) ([^ ]*) ([^ ]*) ([^ ]*):([0-9]*) ([^ ]*)[:-]([0-9]*) ([-.0-9]*) ([-.0-9]*) ([-.0-9]*) (|[-0-9]*) (-|[-0-9]*) ([-0-9]*) ([-0-9]*) \"([^ ]*) (.*) (- |[^ ]*)\" \"([^\"]*)\" ([A-Z0-9-_]+) ([A-Za-z0-9.-]*) ([^ ]*) \"([^\"]*)\" \"([^\"]*)\" \"([^\"]*)\" ([-.0-9]*) ([^ ]*) \"([^\"]*)\" \"([^\"]*)\" \"([^ ]*)\" \"([^\s]+?)\" \"([^\s]+)\" \"([^ ]*)\" \"([^ ]*)\"'
MATCHER = re.compile(REGEX)

AFFINITY = {
    '/api/latest/fleet/download_installer/pkg': 'mac',
    '/api/latest/fleet/download_installer/msi': 'windows',
    '/api/latest/fleet/download_installer/deb': 'linux',
    '/api/latest/fleet/download_installer/rpm': 'linux',
}

ENRICHERS = [
    lambda a: {'geoip': geolite2.lookup(a['client_ip']).to_dict() if geolite2.lookup(a['client_ip']) is not None else None},
    lambda a: {'geohash': geohash.encode(*a['geoip']['location']) if a['geoip'] is not None else None},
    lambda a: {'os_affinity': AFFINITY[a['request_url']['path']] if a['request_url']['path'] in AFFINITY else None},
]

def do_file(bucket, key):
    '''Generates log lines'''
    search = Elasticsearch([os.environ['ES_URL']], serializer=SetEncoder())
    out = []
    response = s3.get_object(Bucket=bucket, Key=key)
    with gzip.GzipFile(fileobj=response["Body"]) as handle:
        for line in handle:
            line = line.decode('utf8')
            match = MATCHER.match(line)
            if not match:
                raise line
            thing = {i[0]: i[1](match.group(n+1)) for n, i in enumerate(fields)}
            thing['_index'] = 'sandbox-prod'
            thing['_id'] = hashlib.sha256(line.encode('utf8')).hexdigest()

            if thing['elb_status_code'] == 200:
                for enricher in ENRICHERS:
                    thing.update(enricher(thing))
                out.append(thing)

    logger.debug(f"Sending {len(out)} items to {os.environ['ES_URL']}")
    bulk(search, out, chunk_size=100)


def lambda_handler(event, _):
    '''Main function'''
    #print("Received event: " + json.dumps(event, indent=2))

    # Get the object from the event and show its content type
    logger.debug(event)
    bucket = event['Records'][0]['s3']['bucket']['name']
    key = urllib.parse.unquote_plus(event['Records'][0]['s3']['object']['key'], encoding='utf-8')
    do_file(bucket, key)
