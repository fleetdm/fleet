# fleetdm.com

This is where the code for the public https://fleetdm.com website lives.


<!-- a [Sails v1](https://sailsjs.com) application -->


## Test locally
Run the following commands to test the site locally:

```sh
npm install -g sails
cd website/
npm install
sails run scripts/build-static-content.js
sails lift
```

Your local copy of the website is now running at [http://localhost:2024](http://localhost:2024)!


## Wipe the production database
I hope you know what you're doing.  The easiest kind of database schema migration:
```sh
sails_datastores__default__url='REAL_DB_URI_HERE' sails run wipe
```

Then when you see the sailboat, hit `CTRL+C` to exit.  All done!


## Bugs
To report a bug or make a suggestion for the website, [click here](https://github.com/fleetdm/fleet/issues).


<!--
### Links

+ [Sails framework documentation](https://sailsjs.com/get-started)
+ [Version notes / upgrading](https://sailsjs.com/documentation/upgrading)
+ [Deployment tips](https://sailsjs.com/documentation/concepts/deployment)
+ [Community support options](https://sailsjs.com/support)
+ [Professional / enterprise options](https://sailsjs.com/enterprise)


### Version info

This app was originally generated on Wed Aug 26 2020 04:48:44 GMT-0500 (Central Daylight Time) using Sails v1.2.5. -->

<!-- Internally, Sails used [`sails-generate@2.0.0`](https://github.com/balderdashy/sails-generate/tree/v2.0.0/lib/core-generators/new). -->

<!--
This project's boilerplate is based on an expanded seed app provided by the [Sails core team](https://sailsjs.com/about) to make it easier for you to build on top of ready-made features like authentication, enrollment, email verification, and billing.  For more information, [drop us a line](https://sailsjs.com/support).

 -->
<!--
Note:  Generators are usually run using the globally-installed `sails` CLI (command-line interface).  This CLI version is _environment-specific_ rather than app-specific, thus over time, as a project's dependencies are upgraded or the project is worked on by different developers on different computers using different versions of Node.js, the Sails dependency in its package.json file may differ from the globally-installed Sails CLI release it was originally generated with.  (Be sure to always check out the relevant [upgrading guides](https://sailsjs.com/upgrading) before upgrading the version of Sails used by your app.  If you're stuck, [get help here](https://sailsjs.com/support).)
-->
