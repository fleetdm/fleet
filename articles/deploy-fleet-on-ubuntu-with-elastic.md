# Deploy Fleet on Ubuntu with Elastic

![Deploy Fleet on Ubuntu with Elastic](../website/assets/images/articles/deploy-fleet-on-ubuntu-with-elastic-1600x900@2x.png)

[<img src="../website/assets/images/articles/deploy-fleet-on-ubuntu-with-elastic-internews_logo-256x237@2x.png" style="width: 128px;"/>](https://internews.org/)

_Today we wanted to feature [Josh](https://defensivedepth.com/), a member of our community. His work was sponsored by [Internews](https://internews.org/). If you are interested in contributing to the Fleet blog, feel free to [contact us](https://fleetdm.com/company/contact) or reach out to [@jdstrong](https://osquery.slack.com/team/U04MTPBAHQS) on the osquery slack._ 

This guide provides a detailed walkthrough for setting up a small production environment of Fleet alongside Elastic components (Elasticsearch, Kibana, Filebeat). The setup integrates Filebeat to collect scheduled query results from Fleet and feed them into Elasticsearch, while Kibana will be utilized for data visualization and the creation of detections. Additionally, Nginx will serve as a reverse proxy for the Kibana and Fleet web interfaces and will segregate the web administration and agent data+control planes of Fleet for more fine-grained access control.

The installation and configuration will begin with the Elastic stack components, followed by Fleet and its dependencies. For this guide, they will all be installed on a single server; however, for larger deployments or requirements of higher availability and scalability, a more distributed approach across multiple servers and geographical regions is recommended.

### Network, server & DNS setup

This guide is based on Ubuntu 22.04 LTS, although the installation procedures for the components remain consistent across newer versions of the operating system.

For this guide, subdomain `fleet.localhost.invalid` is pointed to the server's public IP. Replace this subdomain with a valid one configured as such.

Ports needed, inbound to server:
- `TCP/80` (Only used for the initial Let's Encrypt setup)
- `TCP/443` (Used initially for the Let's Encrypt setup, and then longterm for Fleet distributed agents to checking for data and control)
- `TCP/8443` (Used for Kibana web interface)
- `TCP/9443` (Used for Fleet web interface)

Set up access control where it makes sense - perimeter firewall or on the server itself. Set the ports for the Kibana (`TCP/8443`) and Fleet (`TCP/9443`) web interfaces to only be accessible from a known-trusted IP space. Also set rules for `TCP/443`, which is used for the deployed osquery agents to check in with Fleet. A common configuration is for the web interface ports to be accessible to a single IP or small set of IPs, and for the osquery check in port to be accessible anywhere.

Be aware that if you are using a proxy like Cloudflare, you will need to confirm that the ports in this guide will work as expected.

### Update OS

Let's start by updating the system's packages and creating a workspace directory:

```sh
sudo apt-get update && sudo apt-get dist-upgrade -y
mkdir workspace && cd workspace
```

### Install & configure Certbot

Next up is to install Certbot to create and manage our free Let's Encrypt SSL certificate. This certificate will be used by for all components.

```sh
sudo apt-get install certbot -y
sudo certbot certonly --standalone
```

Select option 1 to spin up a temporary web server. Enter the domain that you have pointed to your public IP. You will need TCP/80 & TCP/443 open to the server.

By default, the certificate and key are saved at:

- Certificate: `/etc/letsencrypt/live/fleet.localhost.invalid/fullchain.pem`
- Key: `/etc/letsencrypt/live/fleet.localhost.invalid/privkey.pem`

### Install & configure Nginx

Let's install Nginx and configure it as a reverse proxy for Fleet and Kibana.

```sh
sudo apt-get install nginx
nano /etc/nginx/sites-available/fleet # use the below config, remember to update the path to the certificate files
sudo ln -s /etc/nginx/sites-available/fleet /etc/nginx/sites-enabled/ # symlink the config file to enable it
nginx -t # Test the config to make sure there are no syntax errors
sudo systemctl reload nginx # Reload nginx to make the config active
sudo systemctl status nginx # Check the reload to confirm that there are no errors
```
Nginx Config file:
```sh
# Define SSL configuration
ssl_certificate /etc/letsencrypt/live/fleet.localhost.invalid/fullchain.pem;
ssl_certificate_key /etc/letsencrypt/live/fleet.localhost.invalid/privkey.pem;

# Common proxy settings
proxy_set_header Host $host;
proxy_set_header X-Real-IP $remote_addr;
proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
proxy_set_header X-Forwarded-Proto $scheme;

# Server block for Kibana on port 8443
server {
	listen 8443 ssl default_server;

	location / {
    	proxy_pass http://localhost:5601;
	}
}

# Server block for Fleet on port 9443 with WebSocket support
server {
	listen 9443 ssl;
	add_header Content-Security-Policy "default-src 'self' 'unsafe-inline' 'unsafe-eval' https: data: blob: wss:; frame-ancestors 'self'";

    location / {
   		proxy_pass https://localhost:4443/;
   		proxy_read_timeout 300;
   		proxy_connect_timeout 300;
   		proxy_set_header Upgrade $http_upgrade;
   		proxy_set_header Connection "Upgrade";
	}
}

# Server block for specific Orbit osquery paths on port 443
server {
	listen 443 ssl;

	location ~* ^/api/(osquery|fleet/orbit/(config|ping)|v1/osquery) {
    	proxy_pass https://localhost:4443;
	}
}
```


### Install & configure Elasticsearch


In case the below does not work, consult Debian package installation instructions at https://www.elastic.co/guide/en/elasticsearch/reference/current/deb.html

Let's download and install Elasticsearch via an Ubuntu package.

One-time prep needed to add the Elastic APT repository:
```sh
wget -qO - https://artifacts.elastic.co/GPG-KEY-elasticsearch | sudo apt-key add -
echo "deb https://artifacts.elastic.co/packages/8.x/apt stable main" | sudo tee -a /etc/apt/sources.list.d/elastic-8.x.list
sudo apt-get update
```

Install the Elasticsearch package (this will install the latest stable version):

```sh
sudo apt-get install elasticsearch
```
The post-install message will contain a password generated for the Elasticsearch built-in superuser (`elastic`). Make note of it as we will need it later.

Enable and start the Elasticsearch service:

```sh
sudo systemctl daemon-reload
sudo systemctl enable --now elasticsearch.service
```

## Install & configure Kibana

Onto Kibana. Let's download, install and do the initial configuration.

```sh
sudo apt-get install kibana
```
Before we start Kibana, we need to edit the configuration file:

```sh
nano /etc/kibana/kibana.yml
```

Set the server host and public base URL by uncommenting and editing the below lines:

```yaml
server.host: "0.0.0.0" # Sets Kibana to listen on all interfaces
server.publicBaseUrl: "https://fleetmd.localhost.invalid:8443" # This should be set to your custom subdomain/port
```

Enable and start the Kibana service:

```sh
sudo /bin/systemctl daemon-reload
sudo /bin/systemctl enable --now kibana.service
```

### Initial configuration

Access Kibana at `https://fleet.localhost.invalid:8443`. If you get stuck at this step, you may not have opened ports 8443 and 9443, as needed in this walkthrough. Generate and enter the initial setup token and the verification code:

```sh
/usr/share/elasticsearch/bin/elasticsearch-create-enrollment-token -s kibana
/usr/share/kibana/bin/kibana-verification-code
```

From there, log in with the username `elastic` and the password that was generated previously, and choose `Explore on my own`. Navigate to `Management` -> `Stack Monitoring` and set up self-monitoring with `set up with self monitoring` and `Turn on monitoring`. This will give you a nice overview of Elasticsearch, Kibana and eventually Filebeat.

## Install & configure Filebeat

The final Elastic component to install is Filebeat. Let's download and configure it to pick up our osquery logs.

```sh
sudo apt-get install filebeat
```

Edit the Filebeat configuration to set up where to send its logs (Elasticsearch). We disable ssl.verification because the connection from Filebeat to Elasticsearch is local (from Filebeat on the server to Elasticsearch on the same system).
Filebeat has built-in support for osquery logs. Let's configure and then enable that filebeat module and then start the Filebeat service:


```sh
sudo nano /etc/filebeat/modules.d/osquery.yml.disabled  # Use the following config
```

```yaml
# Module: osquery

- module: osquery
  result:
    enabled: true

    # Set custom paths for the log files. If left empty,
    # Filebeat will choose the paths depending on your OS.
    var.paths: ["/tmp/osquery_result"]

    # If true, all fields created by this module are prefixed with
    # `osquery.result`. Set to false to copy the fields in the root
    # of the document. The default is true.
    #var.use_namespace: true
```


```sh
sudo filebeat modules enable osquery # Enable the Filebeat osquery module
sudo /bin/systemctl daemon-reload
sudo /bin/systemctl enable --now filebeat.service
```

## Install & configure MySQL

With the Elastic components installed, we can move on to Fleet. First up is installing MySQL and creating the Fleet user and database.

```sh
sudo apt-get install mysql-server -y
mysql -uroot
create database fleet; # This is the database that will be used by Fleet
create user fleet@'localhost' identified by 'FleetPW!'; # Create the mysql user for the Fleet database and set a strong password.
grant all privileges on fleet.* to fleet@'localhost'; # Grant the new user the necessary privileges to the Fleet database.
exit
```

## Install & configure Redis

Redis is used for the Live Query functionality. Let's get it installed.

```sh
sudo apt-get install redis-server -y
```

## Install & configure Fleet

Finally, the linchpin - Fleet. Let's download the latest version. You can find the latest version here: https://github.com/fleetdm/fleet/releases/latest - make sure you download the main Fleet package and not `fleetctl` at this time.

```sh
wget https://github.com/fleetdm/fleet/releases/download/fleet-$VERSION/fleet_$VERSION_linux.tar.gz
tar -xf fleet_v*_linux.tar.gz # Extract the Fleet binary
sudo cp fleet_v*_linux/fleet /usr/bin/ # Copy the the Fleet binary to /usr/bin
fleet version # Sanity check to make sure it runs as expected
```

Next we will create the directory that will contain the config and installers, and create the config itself.

```sh
mkdir /etc/fleet
nano /etc/fleet/fleet.config
```

Use the following as a baseline for your Fleet config:

```yaml
mysql:
  address: 127.0.0.1:3306
  database: fleet
  username: fleet
  password: FleetPW!
redis:
  address: 127.0.0.1:6379
server:
  address: 0.0.0.0:4443
  cert: /etc/letsencrypt/live/fleet.localhost.invalid/fullchain.pem
  key: /etc/letsencrypt/live/fleet.localhost.invalid/privkey.pem
  websockets_allow_unsafe_origin: true # This is needed for Live Query functionality to work with the nginx reverse proxy we are using
```

Next, let's run the `prepare db` command to complete the necessary database prep.

```sh
fleet prepare db --config /etc/fleet/fleet.config
```

### Setup systemd unit file

Now that we are ready to run Fleet, let's create a `systemd` unit file to manage Fleet as a service, and then go ahead and start the service:

```sh
sudo nano /etc/systemd/system/fleet.service # Use the example unit file below
sudo systemctl enable --now fleet.service
sudo systemctl status fleet.service
```

```sh
[Unit]
Description=fleet
After=network.target

[Service]
ExecStart=/usr/bin/fleet serve -c /etc/fleet/fleet.config

[Install]
WantedBy=multi-user.target
```


Finally, complete the Fleet setup via the web interface at https://fleet.localhost.invalid:9443

## fleetctl 

fleetctl is a utility from Fleet that is used to manage Fleet from the command line. Let's download it and get it logged into our instance of Fleet. You can find the latest version here: https://github.com/fleetdm/fleet/releases/latest

```sh
wget https://github.com/fleetdm/fleet/releases/download/fleet-$VERSION/fleetctl_$VERSION_linux.tar.gz
tar -xf fleetctl_*_linux.tar.gz# Extract the fleetct binary
sudo cp fleetctl_v*_linux/fleetctl /usr/bin/ # Copy the the fleetctl binary to /usr/bin
/usr/bin/fleetctl --version # Sanity check to make sure it runs as expected
```

Next, we need to configure it to work with our local instance of Fleet and login to it.

```sh
fleetctl config set --address https://fleet.localhost.invalid::4443
fleetctl login
```

## Generate agents

Fleet ships with support for Orbit, a wrapper around osquery. Orbit makes configuration of osquery much simpler, offers auto-update functionality of osquery as well as additional tables developed by Fleet. In order to install an Orbit/osquery agent, you will need to generate an installer. 

You can start the process of generating Orbit agent packages from the Fleet interface - click on the  `Add Hosts` button. You can generate the packages anywhere that you have `fleetctl`, including on the server itself. Be sure to install the Docker engine if you need to generate installers for Windows.

## Load Fleet standard query library

Fleet has a library of queries that are useful in many different situations - https://fleetdm.com/docs/using-fleet/standard-query-library

Let's go ahead and load them - once this is complete, you can find them in the web interface under Queries.

```sh
git clone https://github.com/fleetdm/fleet.git
cd fleet
fleetctl apply -f docs/01-Using-Fleet/standard-query-library/standard-query-library.yml
```


<meta name="articleTitle" value="Deploy Fleet on Ubuntu">
<meta name="authorGitHubUsername" value="defensivedepth">
<meta name="authorFullName" value="Josh Brower">
<meta name="publishedOn" value="2024-06-12">
<meta name="category" value="guides">
<meta name="description" value="A guide to deploy Fleet and Elastic on Ubuntu.">
<meta name="articleImageUrl" value="../website/assets/images/articles/deploy-fleet-on-ubuntu-with-elastic-1600x900@2x.png">
