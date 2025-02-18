/**
 * @description :: The conventional "custom" hook.  Extends this app with custom server-start-time and request-time logic.
 * @docs        :: https://sailsjs.com/docs/concepts/extending-sails/hooks
 */

module.exports = function defineCustomHook(sails) {

  return {

    /**
     * Runs when a Sails app loads/lifts.
     */
    initialize: async function () {

      sails.log.info('Initializing project hook... (`api/hooks/custom/`)');

      // Check Stripe/Sendgrid configuration (for billing and emails).
      var IMPORTANT_STRIPE_CONFIG = ['stripeSecret', 'stripePublishableKey'];
      var IMPORTANT_SENDGRID_CONFIG = ['sendgridSecret', 'internalEmailAddress'];
      var isMissingStripeConfig = _.difference(IMPORTANT_STRIPE_CONFIG, Object.keys(sails.config.custom)).length > 0;
      var isMissingSendgridConfig = _.difference(IMPORTANT_SENDGRID_CONFIG, Object.keys(sails.config.custom)).length > 0;

      if (isMissingStripeConfig || isMissingSendgridConfig) {

        let missingFeatureText = isMissingStripeConfig && isMissingSendgridConfig ? 'billing and email' : isMissingStripeConfig ? 'billing' : 'email';
        let suffix = '';
        if (_.contains(['silly'], sails.config.log.level)) {
          suffix =
`
> Tip: To exclude sensitive credentials from source control, use:
> • config/local.js (for local development)
> • environment variables (for production)
>
> If you want to check them in to source control, use:
> • config/custom.js  (for development)
> • config/env/staging.js  (for staging)
> • config/env/production.js  (for production)
>
> (See https://sailsjs.com/docs/concepts/configuration for help configuring Sails.)
`;
        }

        let problems = [];
        if (sails.config.custom.stripeSecret === undefined) {
          problems.push('No `sails.config.custom.stripeSecret` was configured.');
        }
        if (sails.config.custom.stripePublishableKey === undefined) {
          problems.push('No `sails.config.custom.stripePublishableKey` was configured.');
        }
        if (sails.config.custom.sendgridSecret === undefined) {
          problems.push('No `sails.config.custom.sendgridSecret` was configured.');
        }
        if (sails.config.custom.internalEmailAddress === undefined) {
          problems.push('No `sails.config.custom.internalEmailAddress` was configured.');
        }

        sails.log.verbose(
`Some optional settings have not been configured yet:
---------------------------------------------------------------------
${problems.join('\n')}

Until this is addressed, this app's ${missingFeatureText} features
will be disabled and/or hidden in the UI.

 [?] If you're unsure or need advice, come by https://sailsjs.com/support
---------------------------------------------------------------------${suffix}`);
      }//ﬁ

      // Set an additional config keys based on whether Stripe config is available.
      // This will determine whether or not to enable various billing features.
      sails.config.custom.enableBillingFeatures = !isMissingStripeConfig;

      // After "sails-hook-organics" finishes initializing, configure Stripe
      // and Sendgrid packs with any available credentials.
      sails.after('hook:organics:loaded', ()=>{

        sails.helpers.stripe.configure({
          secret: sails.config.custom.stripeSecret
        });

        sails.helpers.sendgrid.configure({
          secret: sails.config.custom.sendgridSecret,
          from: sails.config.custom.fromEmailAddress,
          fromName: sails.config.custom.fromName,
        });

        // Validate all values in the githubRepoDRIByPath config variable.
        if(sails.config.custom.githubRepoDRIByPath) {
          if(!_.isObject(sails.config.custom.githubRepoDRIByPath)) {
            throw new Error(`Invalid configuration! An invalid "sails.config.custom.githubRepoDRIByPath" value was provided. If set, this value should be a dictionary, where each key is a path in the GitHub repo, and each value is a GitHub username. Please change this value to be a dictionary and try running this script again.`);
          }
          for(let path in sails.config.custom.githubRepoDRIByPath) {
            if(typeof sails.config.custom.githubRepoDRIByPath[path] !== 'string') {
              throw new Error(`Invalid configuration! A path (${path}) in the "sails.config.custom.githubRepoDRIByPath" config value contains a DRI value that is not a string (type: ${typeof sails.config.custom.githubRepoDRIByPath[path]}). Please change the DRI for this path to be a string containing a single GitHub username and try running this script again.`);
            }
          }
        }

        // Send a request to our Algolia crawler to reindex the website.
        // FUTURE: If this breaks again, use the Platform model to store when the website was last crawled
        // (platform.algoliaLastCrawledWebsiteAt), and then only send a request if it was <30m ago, then remove dyno check.
        if(sails.config.environment === 'production' && process.env.DYNO === 'web.1'){
          sails.helpers.http.post.with({
            url: `https://crawler.algolia.com/api/1/crawlers/${sails.config.custom.algoliaCrawlerId}/reindex`,
            headers: { 'Authorization': sails.config.custom.algoliaCrawlerApiToken}
          }).exec((err)=>{
            if(err){
              sails.log.warn('When trying to send a request to Algolia to refresh the Fleet website search index, an error occurred: '+err);
            }
          });//_∏_
        }
      });//_∏_

      // ... Any other app-specific setup code that needs to run on lift,
      // even in production, goes here ...

    },


    routes: {

      /**
       * Runs before every matching route.
       *
       * @param {Ref} req
       * @param {Ref} res
       * @param {Function} next
       */
      before: {
        '/*': {
          skipAssets: true,
          fn: async function(req, res, next){

            var url = require('url');

            // First, if this is a GET request (and thus potentially a view) or a HEAD request,
            // attach a couple of guaranteed locals.
            if (req.method === 'GET' || req.method === 'HEAD') {

              // The  `_environment` local lets us do a little workaround to make Vue.js
              // run in "production mode" without unnecessarily involving complexities
              // with webpack et al.)
              if (res.locals._environment !== undefined) {
                throw new Error('Cannot attach Sails environment as the view local `_environment`, because this view local already exists!  (Is it being attached somewhere else?)');
              }
              res.locals._environment = sails.config.environment;

              // The `me` local is set explicitly to `undefined` here just to avoid having to
              // do `typeof me !== 'undefined'` checks in our views/layouts/partials.
              // > Note that, depending on the request, this may or may not be set to the
              // > logged-in user record further below.
              if (res.locals.me !== undefined) {
                throw new Error('Cannot attach view local `me`, because this view local already exists!  (Is it being attached somewhere else?)');
              }
              res.locals.me = undefined;
            }//ﬁ

            // Check for query parameters set by ad clicks.
            // This is used to track the reason behind a psychological stage change.
            // If the user performs any action that causes a stage change
            // within 30 minutes of visiting the website from an ad, their psychological
            // stage change will be attributed to the ad campaign that brought them here.
            if(req.param('utm_source') && req.param('creative_id') && req.param('campaign_id')){
              req.session.adAttributionString = `${req.param('utm_source')} ads - ${req.param('campaign_id')} - ${_.trim(req.param('creative_id'), '?')}`;// Trim questionmarks from the end of creative_id parameters.
              // Example adAttributionString: Linkedin - 1245983829 - 41u3985237
              req.session.visitedSiteFromAdAt = Date.now();
            }

            // Check for website personalization parameter, and if valid, absorb it in the session.
            // (This makes the experience simpler and less confusing for people, prioritizing showing things that matter for them)
            // [?] https://en.wikipedia.org/wiki/UTM_parameters
            // e.g.
            //   https://fleetdm.com/device-management?utm_content=mdm
            if (['clear','eo-security', 'eo-it', 'mdm', 'vm'].includes(req.param('utm_content'))) {
              req.session.primaryBuyingSituation = req.param('utm_content') === 'clear' ? undefined : req.param('utm_content');
              // FUTURE: reimplement the following (auto-redirect without querystring to make it prettier in the URL bar), but do it in the client-side JS
              // using whatever that poppushstateblah thing is that makes it so you can change the URL bar from the browser-side code without screwing up
              // the history stack (i.e. back button)
              // ```
              // return res.redirect(req.path);
              // ```
            }//ﬁ

            if (req.method === 'GET' || req.method === 'HEAD') {
              // Include information about the primary buying situation for use in the HTML layout, views, and page scripts.
              res.locals.primaryBuyingSituation = req.session.primaryBuyingSituation || undefined;
            }//ﬁ

            // Next, if we're running in our actual "production" or "staging" Sails
            // environment, check if this is a GET request via some other host,
            // for example a subdomain like `webhooks.` or `click.`.  If so, we'll
            // automatically go ahead and redirect to the corresponding path under
            // our base URL, which is environment-specific.
            // > Note that we DO NOT redirect virtual socket requests and we DO NOT
            // > redirect non-GET requests (because it can confuse some 3rd party
            // > platforms that send webhook requests.)  We also DO NOT redirect
            // > requests in other environments to allow for flexibility during
            // > development (e.g. so you can preview an app running locally on
            // > your laptop using a local IP address or a tool like ngrok, in
            // > case you want to run it on a real, physical mobile/IoT device)
            var configuredBaseHostname;
            try {
              configuredBaseHostname = url.parse(sails.config.custom.baseUrl).host;
            } catch (unusedErr) { /*…*/}
            if ((sails.config.environment === 'staging' || sails.config.environment === 'production') && !req.isSocket && req.method === 'GET' && req.hostname !== configuredBaseHostname) {
              sails.log.info('Redirecting GET request from `'+req.hostname+'` to configured expected host (`'+configuredBaseHostname+'`)...');
              return res.redirect(sails.config.custom.baseUrl+req.url);
            }//•

            // Prevent the browser from caching logged-in users' pages.
            // (including w/ the Chrome back button)
            // > • https://mixmax.com/blog/chrome-back-button-cache-no-store
            // > • https://madhatted.com/2013/6/16/you-do-not-understand-browser-history
            //
            // This also prevents an issue where webpages may be cached by browsers, and thus
            // reference an old bundle file (e.g. dist/production.min.js or dist/production.min.css),
            // which might have a different hash encoded in its filename.  This way, by preventing caching
            // of the webpage itself, the HTML is always fresh, and thus always trying to load the latest,
            // correct bundle files.
            res.setHeader('Cache-Control', 'no-cache, no-store');

            // No session? Proceed as usual.
            // (e.g. request for a static asset)
            if (!req.session) {
              return next();
            }

            // Not logged in? Proceed as usual.
            if (!req.session.userId) { return next(); }

            // Otherwise, look up the logged-in user.
            var loggedInUser = await User.findOne({
              id: req.session.userId
            });

            // If the logged-in user has gone missing, log a warning,
            // wipe the user id from the requesting user agent's session,
            // and then send the "unauthorized" response.
            if (!loggedInUser) {
              sails.log.warn('Somehow, the user record for the logged-in user (`'+req.session.userId+'`) has gone missing....');
              delete req.session.userId;
              return res.unauthorized();
            }

            // Add additional information for convenience when building top-level navigation.
            // (i.e. whether to display "Dashboard", "My Account", etc.)
            if (!loggedInUser.password || loggedInUser.emailStatus === 'unconfirmed') {
              loggedInUser.dontDisplayAccountLinkInNav = true;
            }

            // Expose the user record as an extra property on the request object (`req.me`).
            // > Note that we make sure `req.me` doesn't already exist first.
            if (req.me !== undefined) {
              throw new Error('Cannot attach logged-in user as `req.me` because this property already exists!  (Is it being attached somewhere else?)');
            }
            req.me = loggedInUser;

            // If our "lastSeenAt" attribute for this user is at least a few seconds old, then set it
            // to the current timestamp.
            //
            // (Note: As an optimization, this is run behind the scenes to avoid adding needless latency.)
            var MS_TO_BUFFER = 60*1000;
            var now = Date.now();
            if (loggedInUser.lastSeenAt < now - MS_TO_BUFFER) {
              User.updateOne({id: loggedInUser.id})
              .set({ lastSeenAt: now })
              .exec((err)=>{
                if (err) {
                  sails.log.error('Background task failed: Could not update user (`'+loggedInUser.id+'`) with a new `lastSeenAt` timestamp.  Error details: '+err.stack);
                  return;
                }//•
                sails.log.verbose('Updated the `lastSeenAt` timestamp for user `'+loggedInUser.id+'`.');
                // Nothing else to do here.
              });//_∏_  (Meanwhile...)
            }//ﬁ

            // If this is a GET request, then also expose an extra view local (`<%= me %>`).
            // > Note that we make sure a local named `me` doesn't already exist first.
            // > Also note that we strip off any properties that correspond with protected attributes.
            if (req.method === 'GET') {
              if (res.locals.me !== undefined) {
                throw new Error('Cannot attach logged-in user as the view local `me`, because this view local already exists!  (Is it being attached somewhere else?)');
              }

              // Exclude any fields corresponding with attributes that have `protect: true`.
              var sanitizedUser = _.extend({}, loggedInUser);
              sails.helpers.redactUser(sanitizedUser);

              // If there is still a "password" in sanitized user data, then delete it just to be safe.
              // (But also log a warning so this isn't hopelessly confusing.)
              if (sanitizedUser.password) {
                sails.log.warn('The logged in user record has a `password` property, but it was still there after pruning off all properties that match `protect: true` attributes in the User model.  So, just to be safe, removing the `password` property anyway...');
                delete sanitizedUser.password;
              }//ﬁ

              res.locals.me = sanitizedUser;
              // Create a timestamp of thirty seconds ago. We'll use this to check the age of the user account before creating a fleetwebsite page view record in Salesforce.
              let thirtySecondsAgoAt = Date.now() - (1000 * 30);
              // Start tracking a website page view in the CRM for logged-in users:
              res.once('finish', function onceFinish() {
                // Only track a page view if the requested URL is not a redirect and if this user record is over 30 seconds old (To give time for the background task queued by the signup action to create the initial contact record.
                if(res.statusCode === 200 && sanitizedUser.createdAt < thirtySecondsAgoAt){
                  sails.helpers.flow.build(async ()=>{
                    if(sails.config.environment !== 'production') {
                      sails.log.verbose('Skipping Salesforce integration...');
                      return;
                    }
                    let recordIds = await sails.helpers.salesforce.updateOrCreateContactAndAccount.with({
                      emailAddress: sanitizedUser.emailAddress,
                      firstName: sanitizedUser.firstName,
                      lastName: sanitizedUser.lastName,
                      organization: sanitizedUser.organization,
                      contactSource: 'Website - Sign up',// Note: this is only set on new contacts.
                    });
                    let jsforce = require('jsforce');
                    // login to Salesforce
                    let salesforceConnection = new jsforce.Connection({
                      loginUrl : 'https://fleetdm.my.salesforce.com'
                    });
                    await salesforceConnection.login(sails.config.custom.salesforceIntegrationUsername, sails.config.custom.salesforceIntegrationPasskey);
                    let today = new Date();
                    let nowOn = today.toISOString().replace('Z', '+0000');
                    let websiteVisitReason;
                    if(req.session.adAttributionString && this.req.session.visitedSiteFromAdAt) {
                      let thirtyMinutesAgoAt = Date.now() - (1000 * 60 * 30);
                      // If this user visited the website from an ad, set the websiteVisitReason to be the adAttributionString stored in their session.
                      if(req.session.visitedSiteFromAdAt > thirtyMinutesAgoAt) {
                        websiteVisitReason = this.req.session.adAttributionString;
                      }
                    }
                    // Create the new Fleet website page view record.
                    return await sails.helpers.flow.build(async ()=>{
                      return await salesforceConnection.sobject('fleet_website_page_views__c')
                      .create({
                        Contact__c: recordIds.salesforceContactId,// eslint-disable-line camelcase
                        Account__c: recordIds.salesforceAccountId,// eslint-disable-line camelcase
                        Page_URL__c: `https://fleetdm.com${req.url}`,// eslint-disable-line camelcase
                        Visited_on__c: nowOn,// eslint-disable-line camelcase
                        Website_visit_reason__c: websiteVisitReason// eslint-disable-line camelcase
                      });
                    }).intercept((err)=>{
                      return new Error(`Could not create new Fleet website page view record. Error: ${err}`);
                    });
                  })
                  .exec((err)=>{
                    if(err && typeof err.errorCode !== 'undefined' && err.errorCode === 'DUPLICATES_DETECTED') {
                      // Swallow errors related to duplicate records.
                      sails.log.verbose(`Background task failed: When a logged-in user (email: ${sanitizedUser.emailAddress} visited a page, a Contact/Account/website activity record could not be created/updated in the CRM.`, err);
                    } else if(err){
                      sails.log.warn(`Background task failed: When a logged-in user (email: ${sanitizedUser.emailAddress} visited a page, a Contact/Account/website activity record could not be created/updated in the CRM.`, err);
                    }
                    return;
                  });//_∏_
                }
              });

              // Include information on the locals as to whether billing features
              // are enabled for this app, and whether email verification is required.
              res.locals.isBillingEnabled = sails.config.custom.enableBillingFeatures;
              res.locals.isEmailVerificationRequired = sails.config.custom.verifyEmailAddresses;

              // Include information about the primary buying situation
              // If set in the session (e.g. from an ad), use the primary buying situation for personalization.
              res.locals.primaryBuyingSituation = req.session.primaryBuyingSituation || undefined;

              // * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * //
              // FUTURE: Only show this CTA to users who are below psyStage 6.
              // > The code below is so we don't bother users who have completed the questionnaire

              // Show this logged-in user a CTA to bring them to the /start questionnaire if they do not have billing information saved.
              res.locals.showStartCta = !req.me.hasBillingCard;
              //  * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * //

              // If an expandCtaAt timestamp is set in the user's sesssion, check the value to see if we should expand the CTA.
              if(req.session.expandCtaAt && req.session.expandCtaAt > Date.now()) {
                res.locals.collapseStartCta = true;
              } else {
                res.locals.collapseStartCta = false;
              }
            }//ﬁ

            return next();
          }
        }
      }
    }


  };

};
