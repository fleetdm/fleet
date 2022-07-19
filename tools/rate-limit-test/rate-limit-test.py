import argparse
import logging
import requests 
import time
import uuid
import sys

# Allow users to modify a few portions of the script.
parser = argparse.ArgumentParser(description='A simple script to test rate limiting on a Fleet instance via brute forcing my device pages. Each GET request will contain a new uuid. Rate limiting will start at 720 requests sent in a period of 60 minutes.')
parser.add_argument('--ip', help='Supply the ip of your Fleet instance, default set to https://localhost:8080', default='https://localhost:8080')
parser.add_argument('--requests', help='Specify the number of GET requests sent to the target, default is set to 900', default=900)
#parser.add_argument('--logfile', help='Name of logfile for output', default='rate_limit_output.log')
args = parser.parse_args()

# Suppress some annoying text that warns of no ssl cert being present.
requests.packages.urllib3.disable_warnings(requests.packages.urllib3.exceptions.InsecureRequestWarning)

# Header values that need to be set for this test
headers = {'X-Forwarded-For': '127.0.0.1'}

# Config for a log file that will be created in the directory the script is run from. Will include timestamp, attempt, and http status code
FORMAT = '%(asctime)-15s %(levelname)-8s %(name)s: %(message)s'
logging.basicConfig(
         format=FORMAT,
         level=logging.INFO,
         stream=sys.stdout,
     )
log = logging.getLogger()

# Loop that will attempt a GET request against the Fleet instance using a new uuid for each attempt.
# The sleep(4) is necessary to not hit the spray too many requests at the Fleet instance at once. Doing so will result in 500 codes being returned.
def brute_force():
    for i in range(1, int(args.requests)):
        target = f'{args.ip}/api/latest/fleet/device/{str(uuid.uuid4())}'
        r = requests.get(target, verify=False, headers=headers)
        log.info(f"{i}: {target}: {r.status_code}")
        time.sleep(4)

if __name__ == '__main__':
    # Message displayed after the script is started
    print(f'Rate limit being checked for instance at {args.ip}')
    brute_force()