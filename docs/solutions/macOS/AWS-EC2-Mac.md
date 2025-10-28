# Enrolling Amazon Web Services (AWS) EC2 Mac instances into Fleet

In order to enroll Amazon Web Services (AWS) Elastic Compute Cloud (EC2) Mac instances into Fleet, a few steps are required. These steps will enable automatic enrollment of any instance that's launched from the resulting Amazon Machine Image (AMI)

## 1. First, upload the CloudFormation template to AWS, which will create your AWS Secrets Manager secret.
  * Download the template as text, then upload to CloudFormation. You'll be prompted next for your specific values next.
  * Set `mdmServerDomain` to your Fleet URL beginning with `fleet=` (e.g. `fleet=myfleetserver.example.com`).
  * For API token authentication, set `mdmEnrollmentUser` to `fleet-token` and `mdmEnrollmentPassword` to **your API token**.
  * For username/password authentication, set `mdmEnrollmentUser` to the Fleet server account's email address and `mdmEnrollmentPassword` to the Fleet account's password. The script will use these to authenticate against the Fleet API and receive a token at runtime.
  * `localAdmin` is `ec2-user` by default. `localAdminPassword` can be any value, and will be used during the setup process (i.e. don't make it too tough to type, or use my TypeThis AppleScript to do the typing for you).
## 2. Once the secret is in place, it's time to start a Mac instance to make an Amazon Machine Image (AMI) from. This image will be granted a few privileges allowing the enrollment script to run, and it's all part of this process.
  * Allocate your host and start your Mac instance (anything other than `mac1.metal` can be used to create images that will run on all EC2 Mac instances).
  * Make sure your security group (firewall) has port 5900 open to your IP.
  * Set the **IAM Instance Profile** to the one that was created above. This allows the Mac instance to access the secret at runtime. This privilege can even be auto-removed as an option in the enrollment script.
  * Use the included fleetAutoLogin.sh script in the User Data field. This script will run at launch, set the password of the intended user (creating it if it's not default), and setting automatic login for the account. These settings will ensure automatic enrollment before a user connects.
  * It'll take about 6–20 minutes for the instance to start.
## 3. Connect to the instance via VNC (or Apple's Screen Sharing app, or Apple Remote Desktop). Follow the prompts to allow the script to control System Settings and allow it Accessibility privileges in System Settings as presented.
  * If the prompts don't appear, `osascript /Users/Shared/enroll-ec2-mac.scpt --restart-agent` should bring it up after a minute or so.
  * Once you get the dialog below, **click OK**. That will finish preparing the image for automatic enrollment.
  * (https://raw.githubusercontent.com/aws-samples/amazon-ec2-mac-mdm-enrollment-automation/refs/heads/main/SetupComplete.png)
  * After clicking OK, in the **AWS console**, go to **EC2**, then to **Instances**, then the intended Mac instance. From the **Actions** menu in the upper right, click **Image and templates**, and then **Create image**.
  * Though it's officially better to have it reboot as part of the process, I've made many images with "No reboot" checked and without issue. If you leave reboot on, the Mac will automatically enroll into Fleet when it restarts.
  * * This is what I term a "console reboot," which reboots the underlying AWS Nitro system and takes about 10-15 minutes. If you reboot a Mac instance normally from the UI (or `sudo reboot` et. al.), it's the typical ~ 1 minute.
  * Imaging may take some time to complete (I've had anywhere from 5 minutes to an hour for a 100GB EBS volume), it's basically locking the bits as soon as you hit that **Create image** button.
## 4. Launch a new Mac instance using the new image.
  * No user data script required after image setup, other than a defaults write to set the region for the secret (if it's different to the one the instance is launching in).
  * IAM profile must be set as above.
  * The Mac will take the same 6-20 minutes to launch, and will appear enrolled in the Fleet console shortly after.
