# fleetdm.com

This is where the code for the public https://fleetdm.com website lives.


## Bugs
To report a bug or make a suggestion for the website, [click here](https://github.com/fleetdm/fleet/issues).

## Testing locally

See https://fleetdm.com/handbook/digital-experience#test-fleetdm-com-locally

## Deploying the website
To deploy changes to the website to production, merge changes to the `main` branch.  If the changes affect the website's code, or touch any files that the website relies on to build content, such as the query library, osquery schema, docs, handbook, articles, etc., then the website will be redeployed.

> Wondering how this works?  This is implemented in a GitHub action in this repo.  Check out the code there to see how it works!  For help understanding what `sails run` and `npm run` commands in there do, check the scripts in `website/package.json` and in `website/scripts/`.


### Changing the database schema
To deploy new code to production that relies on changes to the database schema or other external systems (e.g. Stripe), first put the website in "maintenance mode" in Heroku.  Then, make your changes in the database schema.   Next, if you have a script to fix/migrate existing data, go ahead and run it now.  (e.g. `sails run fix-or-migrate-existing-data`).  Then, merge your changes and wait for the deploy to finish.  Finally, switch off "maintenance mode" in Heroku.

Note that entering maintenance mode prevents visitors from using the website, so it should be used sparingly, and ideally at low-traffic times of day.

> Warning: Doing an especially sensitive schema migration?  There is a potential timing issue to consider, thanks to an infrastructure change that eliminated downtime during deploys by using Heroku's built-in support for hot-swapping.  Read more in https://github.com/fleetdm/fleet/issues/6568#issuecomment-1211503881

### Wiping the production database
I hope you know what you're doing.  The "easiest" kind of database schema migration:
```sh
sails_datastores__default__url='REAL_DB_URI_HERE' sails run wipe
```

Then when you see the sailboat, hit `CTRL+C` to exit.  All done!



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
