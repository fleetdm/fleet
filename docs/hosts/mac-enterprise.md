# Enrolling multiple Macs

If you're managing an enterprise environment with multiple Mac devices, you likely have an enterprise deployment tool like [Munki](https://www.munki.org/munki/) or [Jamf Pro](https://www.jamf.com/products/jamf-pro/) to deliver software to your mac. You can deploy osqueryd and enroll all your macs into kolide using your software management tool of choice. 

First, [download](https://osquery.io/downloads/) and import the osquery package into your software management repository. You can also use the community supported autopkg [recipe](https://github.com/autopkg/keeleysam-recipes/tree/master/osquery)
to keep osqueryd updated. 


Next, you will have to create an enrollment package to get osqueryd running and talking to kolide. Here, you'll have to create a custom package because you have to provide specific information about your kolide setup. We created a Makefile to help you build a macOS enrollment package. 

First, download the kolide repository from Github and navigate to the `tools/mac` directory. 

Next, you'll have to edit the `config.mk` file. You'll find all the necessary information by clicking "Add New Host" in your kolide server.

 - Set the `KOLIDE_HOSTNAME` variable to the FQDN of your kolide server.
 - Set the `ENROLL_SECRET` variable to the enroll secret you got from kolide.
 - Paste the contents of the kolide TLS certificate after the following line:
      ```
      define KOLIDE_TLS_CERTIFICATE
      ``` 

Note that osqueryd requires a full certificate chain, even for certificates which might be trusted by your keychain. The "Fetch Kolide Certificate" button in the Add New Host screen will attempt to fetch the full chain for you. 

Once you've configured the `config.mk` file with the corect variables, you can run `make` in the `tools/mac` directory. Running `make` will create a new `kolide-enroll.pkg` file which you can import into your software repository and deploy to your macs. 

The enrollment package must installed after the osqueryd package, and will install a LaunchDaemon to keep the osqueryd process running.

