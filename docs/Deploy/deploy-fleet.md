# Deploy Fleet

Just need to kick the tires? Create an account to [Try Fleet locally](https://fleetdm.com/try-fleet) on your device. You can deploy FleetDM on your own servers or virtual machines, which involves manually setting up the necessary components, such as the FleetDM server, database, and any dependencies. This approach, although more lengthy than a Cloud deployment, gives you full control over the environment and configuration.
A technical installation guide sponsored by Internews for a self-contained single server deployment of FleetDM on Ubuntu, with a few Elastic Stack components, can be found [here](https://fleetdm.com/guides/deploy-fleet-on-ubuntu-with-elastic).

On the other hand, use our easy steps below to deploy to Render for an easy one-click proof of concept. Or, choose AWS with Terraform to deploy at scale.

<div purpose="deploying-guide-buttons" class="d-flex flex-md-row flex-column">
    <a href="#render">
        <div>
            <img src="/images/docs/render-logo-147x80@2x.png">
            <p>Deploy to Render in 5 minutes</p>
        </div>
    </a>
    <a href="#aws">
        <div>
        <img src="/images/docs/aws-logo-133x80@2x.png">
        <p>Scale on AWS with Terraform</p>
        </div>
    </a>
</div>


<h2 class="d-none markdown-heading">Render</h2>
<h2 id="render">Deploy to Render in 5 minutes</h2>


Render is a cloud hosting service that makes it easy to get up and running fast, without the typical configuration headaches of larger enterprise hosting providers.

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/hly0tAOqveA?rel=0" frameborder="0" allowfullscreen></iframe>
</div>

### Prerequisites

- A Render account with payment information.

>The Fleet Render Blueprint will provision a web service, a MySQL database, and a Redis in-memory data store. Each service requires Render's standard plan at a cost of **$7/month** each, totaling **$21/month**.


### Instructions

<div purpose="deploy-to-render-button">
    <a href="https://render.com/deploy?repo=https://github.com/fleetdm/fleet">
        <img src="https://render.com/images/deploy-to-render-button.svg" alt="Deploy to Render">
    </a>
</div>

1. Click "Deploy to Render" to open the Fleet Blueprint on Render. You will be prompted to create or log in to your Render account with associated payment information.

2. Give the Blueprint a unique name like `yourcompany-fleet`.

3. Click "**Apply.**" Render will provision your services, which should take less than five minutes. 

4. Click the "**Dashboard**" tab in Render when provisioning is complete to see your new services. 

5. Click on the "**Fleet**" service to reveal the Fleet URL.

6. Click on the URL to open your Fleet instance, then follow the on-screen instructions to set up your Fleet account.

Support for add/install software features is coming soon. Get [commmunity support](https://chat.osquery.io/c/fleet).

<h2 class="d-none markdown-heading">AWS</h2>
<h2 id="aws">Deploy at scale with AWS and Terraform</h2>

The simplest way to get started with Fleet at scale is to use AWS with Terraform.

This workflow takes about 30 minutes to complete and supports between 10 and 350,000 hosts.

### Prerequisites

- A new or existing Amazon Web Services (AWS) account

- An AWS Identity and Access Management (IAM) user with administrator privileges

- The latest version of AWS Command Line Interface `awscli`

- The latest version of HashiCorp Terraform

- A Fully-Qualified Domain Name (FQDN) for hosting Fleet

### Instructions

1. [Download](https://github.com/fleetdm/fleet/blob/main/terraform/example/main.tf) the Fleet `main.tf` Terraform file.

2. Edit the following variables in the `main.tf` Terraform file you just downloaded to match your environment:
    
    ```
    # Change these to match your environment.
    domain_name = "fleet.example.com"
    vpc_name = "fleet-vpc"
    osquery_carve_bucket_name   = "fleet-osquery-carve"
    osquery_results_bucket_name = "fleet-osquery-results"
    osquery_status_bucket_name  = "fleet-osquery-status"
    ```

    > Terraform modules for Fleet features can be enabled and disabled by commenting or uncommenting sections of the code as needed. To learn more about the modules, check out our [AWS with Terraform advanced guide](https://fleetdm.com/docs/deploy/deploy-on-aws-with-terraform).

3. Log in to [your AWS account](https://aws.amazon.com/iam/) using your IAM identity.

4. Run a command like the following in Terminal:
    
    ```
    % terraform init ~/Downloads/main.tf
    ```

    > If the file was not downloaded to the downloads folder, ensure that you adjust the file path in the command.

    > This step will take around 15 minutes.

5. Run the following commands in Terminal:

    ```
    terraform apply -target module.fleet.module.vpc
    ```

6. Next, run this command:
    
    ```
    terraform apply -target module.osquery-carve -target module.firehose-logging
    ```

7. Log in to your AWS Route 53 instance

8. Run the following commands in Terminal:
    ```
    terraform apply -target aws_route53_zone.main
    ```

9. From the Terminal output, obtain the NS records created for the zone and add them to the parent DNS zone in the AWS Route 53 GUI.

10. Run the following commands in Terminal:
    
    ```
    terraform apply -target module.fleet
    ```

11. Lastly, run the following in Terminal:
    
    ```
    terraform apply
    ```

12. That’s it! Next, you can skip ahead and already log in to Fleet to [enroll a host](https://fleetdm.com/docs/using-fleet/enroll-hosts), or you can continue with our other Deployment documents for other options that might fit your environment:
- [Single sign-on (SSO)](https://fleetdm.com/docs/deploy/single-sign-on-sso)
- [Upgrading Fleet](https://fleetdm.com/docs/deploy/upgrading-fleet)
- [Reference architectures](https://fleetdm.com/docs/deploy/reference-architectures)

<meta name="pageOrderInSection" value="100">
<meta name="description" value="Learn how to easily deploy Fleet on Render or AWS with Terraform.">
