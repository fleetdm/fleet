module.exports = {


  friendlyName: 'Receive Zapier events',


  description: 'Receive events from Zapier.',


  inputs: {
    eventName: {
      type: 'string',
      description: 'The unique identifier for this Zap.',
      moreInfoUrl: 'https://zapier.com/app/assets/zaps/folders/2035513',
      required: true,
    },
    data: {
      type: {},
      description: 'Data associated with this event.',
      whereToGet: { description: 'Check out the Zap in question and see what it\'s sending via HTTP.' },
      required: true,
    },
    webhookSecret: {
      type: 'string',
      description: 'Used to verify that requests are coming from where we think they are.',
      required: true,
    },
  },


  exits: {
    success: { description: 'An event has successfully been received.' },
    unrecognizedEventName: { description: 'I do not know how to handle that kind of event.', responseType: 'ok' },// TODO: how will zapier react to receiving a bad request response?
    couldNotMatchLinkedinId: { description: 'A linkedIn company could not be found using the provided linkedIn url', responseType: 'ok' }
  },


  fn: async function ({eventName, data, webhookSecret}) {
    let assert = require('assert');

    if (!sails.config.custom.zapierWebhookSecret) {
      throw new Error('No webhook secret configured!  (Please set `sails.config.custom.zapierWebhookSecret`.)');
    }

    if (!sails.config.custom.iqSecret) {
      throw new Error('No iqSecret configured!  (Please set `sails.config.custom.iqSecret`.)');
    }

    if (sails.config.custom.zapierWebhookSecret !== webhookSecret) {
      throw new Error('Received unexpected webhook request with webhookSecret set to: '+webhookSecret);
    }

    // Zap: https://zapier.com/editor/281086063     // «« TODO: actually publish this Zap when deployed and ready
    if (eventName === 'receive-new-customer-data') {
      assert(_.isObject(data));
      assert(_.isString(data.newMarketingStage));
      assert(_.isString(data.name));
      assert(_.isString(data.website));
      assert(_.isString(data.linkedinCompanyPageUrl));
      assert(_.isString(data.persona) && AdCampaign.validate('persona', data.persona));

      // Enrich to obtain linkedin company ID using provided data.
      // Remove any trailing slashes from the LinkedIn URL.
      let trailingSlashlessLinkedinCompanyUrl = _.trim(data.linkedinCompanyPageUrl, '/');
      // Split the LinkedIn url by slashes
      let splitLinkedinCompanyUrl = trailingSlashlessLinkedinCompanyUrl.split('/');
      // Grab the last fragment of the URL, we'll use this for the coreSignal API request to
      let linkedinCompanyIdOrSLug = splitLinkedinCompanyUrl[splitLinkedinCompanyUrl.length - 1];
      let matchedCompanyPageInfo = await sails.helpers.http.get('https://api.coresignal.com/cdapi/v1/linkedin/company/collect/'+linkedinCompanyIdOrSLug, {}, {
        Authorization: `Bearer ${sails.config.custom.iqSecret}`,
        'content-type': 'application/json'
      }).intercept((err)=>{
        sails.log.info(`When the receive-from-zapier webhook received a request about a Salesforce record, a linkedin company could not be found using the provided linkedIn URL (${data.linkedinCompanyPageUrl})`, err);
        return 'couldNotMatchLinkedinId';
      });

      // (FUTURE: make field for this and have it already in CRM so this step isn't necessary)
      let linkedinCompanyId = matchedCompanyPageInfo.id;

      // Check if we have enough space in our current active campaign.
      // If so, then use it and update its inventory.  Otherwise, prepare to create
      // a new campaign, and use that instead, updating our set of active campaigns
      // in the db, including marking the new one as the latest and greatest.  Along the way,
      // communicate with Campaign Manager to update or create the appropriate campaign.
      let latestCampaign = await AdCampaign.findOne({ isLatest: true, persona: data.persona });
      if (latestCampaign && latestCampaign.linkedinCompanyIds.length < 100) {
        // Update ad campaign in Campaign Manager
        // > For help w/ Linkedin API, see https://github.com/fleetdm/confidential/tree/main/ads
        let filterCriteriaForLatestCampaign = latestCampaign.linkedinCompanyIds.map((id)=>{
          return `urn:li:organization:${id}`;
        });
        await sails.helpers.http.sendHttpRequest.with({
          method: 'POST',
          url: 'TODO',// TODO: create zap to updates an existing campaign.
          data: {
            targetingCriteria: filterCriteriaForLatestCampaign,
          }
          // TODO: call out to a "update LI campaign" Zap via HTTP, which then talks to linkedin because we don't have access to talk to the linkedin api directly
          // See the other example below of "create LI campaign" below for example of fields to send in  (see also the other repo)
        }).retry();

        await AdCampaign.updateOne({ id: latestCampaign.id }).set({
          linkedinCompanyIds: _.uniq(latestCampaign.linkedinCompanyIds.concat(linkedinCompanyId))
        });
      } else {

        // First, mark the old campaign as no longer the latest.
        // Note: Since we might not have done the first-time setup for
        // this persona yet, it's possible there won't actually be a
        // campaign record yet.  (In that case, we'll create it momentarily.)
        if (latestCampaign) {
          await AdCampaign.updateOne({ id: latestCampaign.id }).set({
            isLatest: false,
          });
        }//ﬁ

        // Create a placeholder linkedinCampaignUrn value to create the record with initially
        // We'll use this value in a subsequent webhook run that will save update the record with the real linkedinCampaignUrn (once it has been created).
        let placeholderUrn = sails.helpers.strings.random();
        let nowAt = new Date();
        let newCampaignName = `${nowAt.toIsoString().trim('T')[0]} @ ${nowAt.toLocaleString().split(', ')[1]}`
        // Now save an incomplete reference to the new LinkedIn campaign.
        latestCampaign = await AdCampaign.create({
          isLatest: true,
          persona: data.persona,
          linkedinCampaignUrn: placeholderUrn,
          linkedinCompanyIds: [ linkedinCompanyId ],
        }).fetch();

        // TODO: call out to a "create campaign" Zap via HTTP, which then talks to linkedin because we don't have access to talk to the linkedin api directly
        // Then create new ad campaign in Campaign Manager
        // > For help w/ Linkedin API, see https://github.com/fleetdm/confidential/tree/main/ads
        let report = await sails.helpers.http.sendHttpRequest.with({
          method: 'POST',
          url: 'https://hooks.zapier.com/hooks/catch/3627242/2wdx23r/',
          headers: {
            'X-Custom-Secret-For-Zapier-LinkedIn-Proxy': sails.config.custom.zapierWebhookSecret
          },
          body: {
            campaignGroup: sails.config.custom.linkedinAbmCampaignGroupUrn,
            name: newCampaignName,
            targetingCriteria: [`urn:li:organization:${linkedinCompanyId}`],
            placeholderUrn,
          },
        }).retry();

        assert(_.isString(report.linkedinCampaignUrn) && '' !== report.linkedinCampaignUrn);

      }
      // Zap: https://zapier.com/editor/280954803     // «« TODO: Add a webhook request to the zap with this event name and data.
    } else if(eventName === 'update-placeholder-campaign-urn') {
      assert(_.isObject(data));
      assert(_.isString(data.placeholderUrn));
      assert(_.isString(data.linkedinCampaignUrn));

      let adCampaignWithThisPlaceholderUrn = await AdCampaign.findOne({linkedinCampaignUrn: data.placeholderUrn})
      if(!adCampaignWithThisPlaceholderUrn) {
        sails.log.warn(`when the receive-from-zapier webhook received an event to update an AdCampaign record with a non-placeholder linkedinCampaignUrn value (${data.linkedinCampaignUrn}), no record could be found with the specified placeholder (${data.placeholderUrn}).`);
      }
      await AdCampaign.updateOne({linkedinCampaignUrn: data.placeholderUrn}).set({
        linkedinCampaignUrn: data.linkedinCampaignUrn
      });
    } else {
      throw 'unrecognizedEventName';
    }

  }


};
