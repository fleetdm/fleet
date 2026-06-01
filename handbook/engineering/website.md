# fleetdm.com

This handbook page details the processes for maintaining and releasing changes to fleetdm.com.


## QA a change to fleetdm.com

Each PR to the website is manually checked for quality and tested before going live on fleetdm.com. To test any change to fleetdm.com:

1. Write clear step-by-step instructions to confirm that the change to the fleetdm.com functions as expected and doesn't break any possible automation. These steps should be simple and clear enough for anybody to follow.

2. [View the website locally](#test-fleetdm-com-locally) and follow the QA steps in the request ticket to test changes.

3. Check the change in relation to all breakpoints and [browser compatibility](#check-browser-compatibility-for-fleetdm-com), Tests are carried out on [supported browsers](https://fleetdm.com/docs/using-fleet/supported-browsers) before website changes go live.


## Test fleetdm.com locally 

When making changes to the Fleet website, you can test your changes by running the website locally. To do this, you'll need the following:

- A local copy of the [Fleet repo](https://github.com/fleetdm/fleet).
- [Node.js](https://nodejs.org/en/download/)
- (Optional) [Sails.js](https://sailsjs.com/) installed globally on your machine (`npm install sails -g`)

Once you have the above follow these steps:

1. Open your terminal program, and navigate to the `website/` folder of your local copy of the Fleet repo.
    
    > Note: If this is your first time running this script, you will need to run `npm install` inside of the website/ folder to install the website's dependencies.


2. Run the `build-static-content` script to generate HTML pages from our Markdown and YAML content.
  - **With Node**, you will need to use `node ./node_modules/sails/bin/sails run build-static-content` to execute the script.
  - **With Sails.js installed globally** you can use `sails run build-static-content` to execute the script.

    > When this script runs, the website's configuration file ([`website/.sailsrc`](https://github.com/fleetdm/fleet/blob/main/website/.sailsrc)) will automatically be updated with information the website uses to display content built from Markdown and YAML. Changes to this file should never be committed to the GitHub repo. If you want to exclude changes to this file in any PRs you make, you can run this terminal command in your local copy of the Fleet repo: `git update-index --assume-unchanged ./website/.sailsrc`.
    
    > Note: You can run `npm run start-dev` in the `website/` folder to run the `build-static-content` script and start the website server with a single command.

3. Once the script is complete, start the website server:
  - **With Node.js:** start the server by running `node ./node_modules/sails/bin/sails lift`
  - **With Sails.js installed globally:** start the server by running `sails lift`.

4. When the server has started, the Fleet website will be available at [http://localhost:2024](http://localhost:2024)
    
  > **Note:** Some features, such as self-service license dispenser and account creation, are not available when running the website locally. If you need help testing features on a local copy, `@`mention `eashaw` in the [#g-website](https://fleetdm.slack.com/archives/C058S8PFSK0) Slack channel.


## Check production dependencies of fleetdm.com

Every week, we run `npm audit --only=prod` to check for vulnerabilities on the production dependencies of fleetdm.com. Once we have a solution to configure GitHub's Dependabot to ignore devDependencies, this [manual process](https://www.loom.com/share/153613cc1c5347478d3a9545e438cc97?sid=5102dafc-7e27-43cb-8c62-70c8789e5559) can be replaced with Dependabot.


## Triage and address vulnerabilities in the `website/` code base

When Dependabot or code scanning surfaces critical or high-severity vulnerabilities in the `/website` directory:

1. **Filter out development-only dependencies.** Dismiss any alerts for packages that are only used during development and never ship to production. When dismissing, include a message with proof, e.g.:
   > devdep, unused in prod. Proof:
   > https://github.com/fleetdm/fleet/blob/3a6ecb5a11fdbdf290faf7fdd7ffa6b29335892f/website/package-lock.json#L10798 _(link to the relevant line)_

2. **Assess real-world applicability.** Some vulnerabilities only apply under specific conditions (e.g., a path-to-regex vulnerability that only triggers with 3+ dynamic path params, which fleetdm.com doesn't use). Note these for upstream fixes but deprioritize if not exploitable in our setup.

3. **Address real vulnerabilities.** For confirmed production-impacting vulnerabilities:
   - Identify the root cause (e.g., a transitive dependency using a `~` semver range instead of `^`).
   - Publish patch releases of affected upstream packages (e.g., `@sailshq/router`, `sails-hook-organics`) as needed.
   - Regenerate the lockfile in `fleetdm/fleet:website` after upstream fixes are published.

4. **Reference video walkthrough.** For a detailed walkthrough of this process, see [this confidential video](https://drive.google.com/file/d/17JF1jtEjVc7wkeXYA-2GIJbh9GDPWJEc/view?usp=sharing) (accessible to fleeties only).


## Respond to a 5xx error on fleetdm.com

Production systems can fail for various reasons, and it can be frustrating to users when they do, and customer experience is significant to Fleet. In the event of system failure, Fleet will:
- investigate the problem to determine the root cause.
- identify affected users.
- escalate if necessary.
- understand and remediate the problem.
- notify impacted users of any steps they need to take (if any).  If a customer paid with a credit card and had a bad experience, default to refunding their money.
- Conduct an incident post-mortem to determine any additional steps we need (including monitoring) to take to prevent this class of problems from happening in the future.


## Check browser compatibility for fleetdm.com

A [browser compatibility check](https://www.loom.com/share/4b1945ccffa14b7daca8ab9546b8fbb9?sid=eaa4d27a-236b-426d-a7cb-9c3bdb2c8cdc) of [fleetdm.com](https://fleetdm.com/) should be carried out monthly to verify that the website looks and functions as expected across all [supported browsers](https://fleetdm.com/docs/using-fleet/supported-browsers).

- We use [BrowserStack](https://www.browserstack.com/users/sign_in) (logins can be found in [1Password](https://start.1password.com/open/i?a=N3F7LHAKQ5G3JPFPX234EC4ZDQ&v=3ycqkai6naxhqsylmsos6vairu&i=nwnxrrbpcwkuzaazh3rywzoh6e&h=fleetdevicemanagement.1password.com)) for our cross-browser checks.
- Check for issues against the latest version of Google Chrome (macOS). We use this as our baseline for quality assurance.
- Document any issues in GitHub as a [bug](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=bug%2C%3Areproduce&template=bug-report.md&title=), and assign them for fixing.
- If in doubt about anything regarding design or layout, please reach out to the [Head of Design](https://fleetdm.com/handbook/product-design#team).


## Check for new versions of osquery schema

When a new version of osquery is released, the Fleet website needs to be updated to use the latest version of the osquery schema. To do this, we update the website's `versionOfOsquerySchemaToUseWhenGeneratingDocumentation` configuration variable in [website/config/custom.js](https://github.com/fleetdm/fleet/blob/6eb6884c4f02dc24b49f394abe9dde5fd1875c55/website/config/custom.js#L327). The osquery schema is combined with Fleet's [YAML overrides](https://github.com/fleetdm/fleet/tree/main/schema/tables) to generate the [JSON schema](https://github.com/fleetdm/fleet/blob/main/schema/osquery_fleet_schema.json) used by the table schema side panel in Fleet, as well as fleetdm.com's [table schema](/tables).

> Note: The version number used in the `versionOfOsquerySchemaToUseWhenGeneratingDocumentation` variable must correspond to a version of the JSON osquery schema in the [osquery/osquery-site repo](https://github.com/osquery/osquery-site/tree/main/src/data/osquery_schema_versions).


## Restart Algolia manually

At least once every hour, an Algolia crawler reindexes the Fleet website's content. If an error occurs while the website is being indexed, Algolia will block our crawler and respond to requests with this message: `"This action cannot be executed on a blocked crawler"`.

When this happens, you'll need to manually start the crawler in the [Algolia crawler dashboard](https://crawler.algolia.com/admin/) to unblock it. 
You can do this by logging into the crawler dashboard using the login saved in 1password and clicking the "Restart crawling" button on our crawler's ["overview" page](https://crawler.algolia.com/admin/crawlers/497dd4fd-f8dd-4ffb-85c9-2a56b7fafe98/overview).

No further action is needed if the crawler successfully reindexes the Fleet website. If another error occurs while the crawler is running, take a screenshot of the error and add it to the GitHub issue created for the alert and @mention `eashaw` for help.


## Change the "Integrations admin" Salesforce account password

Salesforce requires that the password to the "Integrations admin" account is changed every 90 days. When this happens, the Salesforce integrations on the Fleet website/Hydroplane will fail with an `INVALID_LOGIN` error. To prevent this from happening:

1. Log into the "Integrations admin" account in Salesforce.
2. Change the password and save it in the shared 1Password vault.
3. Request a new security token for the "Integrations admin" account by clicking the profile picture » `Settings` » `Reset my security token` (This will be sent to the email address associated with the account).
4. Update the `sails_config__custom_salesforceIntegrationPasskey` config variable in Heroku to be `[password][security token]` (For both the Fleet website and Hydroplane).


## Re-run the "Deploy Fleet Website" action

If the action fails, please complete the following steps:
1. Head to the fleetdm-website app in the [Heroku dashboard](https://heroku.com) and select the "Activity" tab.
2. Select "Roll back to here" on the second to most recent deploy.
3. Head to the fleetdm/fleet GitHub repository and re-run the Deploy Fleet Website action.


## Respond to website code scanning alerts

Every week, the website maintainer looks for any new [code scanning alerts](https://github.com/fleetdm/fleet/security/code-scanning) that have been created for the `website/` folder. If any are found they:
1. Determine if the alert is affecting the production evironment. If this is an alert for a vulnerability in a depedency, then the maintainer will look at what brings the depedency into the codebase.
2. Respond to the alert. 
   - If the alert is for code that has been merged into the repo, the maintiner will create a pull request to fix it. 
   - If the alert is for a dependency that runs in production, the maintainer will upgrade it to a version that is not affected by the vulnerability. 
   - If the alert is for a devDependency or a dependency of a devDependency, the maintainer will dismiss the alert as a false-positive, because it does not affect the production environment.

## Incubate website dependency changes

Pull requests that modify `website/package.json` or `website/package-lock.json` must wait 72 hours after the most recent commit to either file before they can merge to `main`. This incubation period gives the maintainer time to spot regressions or supply-chain concerns introduced by a dependency bump before it reaches the website's production environment.

The `Incubate website dependency changes` GitHub Actions workflow enforces this as a required status check. The check runs on every PR that touches those files and re-evaluates open PRs nightly, once 72 hours have elapsed the check turns green automatically. Pushing a new commit to either file resets the clock.


<meta name="maintainedBy" value="lukeheath">
<meta name="description" value="Processes for maintaining and releasing changes to fleetdm.com.">
