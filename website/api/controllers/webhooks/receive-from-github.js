module.exports = {


  friendlyName: 'Receive from GitHub',


  description: 'Receive webhook requests and/or incoming auth redirects from GitHub.',


  extendedDescription: 'Useful for automation, visibility of changes, and abuse monitoring.',


  inputs: {
    botSignature: { type: 'string', },
    action: { type: 'string', example: 'opened', defaultsTo: 'ping', moreInfoUrl: 'https://developer.github.com/v3/activity/events/types' },
    sender: { required: true, type: {}, example: {login: 'johnabrams7'} },
    // Org-level webhooks may not include a `repository` object.
    repository: { type: {}, example: {name: 'fleet', owner: {login: 'fleetdm'}} },
    changes: { type: {}, description: 'Only present when webhook request is related to an edit on GitHub.' },
    issue: { type: {} },
    comment: { type: {} },
    pull_request: { type: {} },//eslint-disable-line camelcase
    label: { type: {} },
    release: { type: {} },
    projects_v2_item: { type: {} }, //eslint-disable-line camelcase
  },


  fn: async function ({botSignature, action, sender, repository, changes, issue, comment, pull_request: pr, label, release, projects_v2_item: projectsV2Item}) {

    // Grab the set of GitHub pull request numbers the bot considers "unfrozen" from the platform record.
    // If there is more than one platform record, or it is missing, we'll throw an error.
    let platformRecords = await Platform.find();
    let platformRecord = platformRecords[0];
    if(!platformRecord) {
      throw new Error(`Consistency violation: when the GitHub webhook received an event, no platform record was found.`);
    } else if(platformRecords.length > 1) {
      throw new Error(`Consistency violation: when the GitHub webhook received an event, more than one platform record was found.`);
    }

    // let pocketOfPrNumbersUnfrozen = platformRecord.currentUnfrozenGitHubPrNumbers;
    // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

    // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
    // let IS_FROZEN = false;// « Set this to `true` whenever a freeze is in effect, then set it back to `false` when the freeze ends.
    // > ^For context on the history of this bit of code, which has gone been
    // > implemented a couple of different ways, and gone back and forth, check out:
    // > https://github.com/fleetdm/fleet/pull/5628#issuecomment-1196175485
    // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

    let GITHUB_USERNAMES_OF_BOTS_AND_MAINTAINERS = [// « Used in multiple places below.
      // FUTURE: move this array into website/config/custom.js alongside the other similar config
      // and reference here as e.g. `sails.config.custom.githubUsernamesOfBotsAndMaintainers`

      // Bots
      'vercel[bot]',
      'fleet-release',

      // Humans
      'noahtalerman',
      'lppepper2',
      'mike-j-thomas',
      'mikermcneil',
      'lukeheath',
      'zwass',
      'rachelelysia',
      'gillespi314',
      'mna',
      'edwardsb',
      'eashaw',
      'lucasmrod',
      'ksatter',
      'hollidayn',
      'ghernandez345',
      'rfairburn',
      'zayhanlon',
      'alexmitchelliii',
      'sampfluger88',
      'ireedy',
      'mostlikelee',
      'AnthonySnyder8',
      'jahzielv',
      'getvictor',
      'phtardif1',
      'pintomi1989',
      'nonpunctual',
      'dantecatalfamo',
      'PezHub',
      'SFriendLee',
      'ddribeiro',
      'allenhouchins',
      'harrisonravazzolo',
      'tux234',
      'ksykulev',
      'jmwatts',
      'mason-buettner',
      'iansltx',
      'sgress454',
      'BCTBB',
      'kc9wwh',
      'JordanMontgomery',
      'bettapizza',
      'irenareedy',
      'jakestenger',
      'AndreyKizimenko',
      'MagnusHJensen',
      'MunkiMind',
      'spalmesano0',
      'escomeau',
      'cdcme',
      'kevinmalkin12',
      'karmine05',
      'ericswenson0',
      'kitzy',
      'Seedity',
      'NickBlee',
      'GrayW',
    ];

    let GREEN_LABEL_COLOR = 'C2E0C6';// « Used in multiple places below.  (FUTURE: Use the "+" prefix for this instead of color.  2022-05-05)

    let GITHUB_USERNAME_OF_DRI_FOR_LABELS = 'noahtalerman';// « Used below (FUTURE: Remove this capability as Fleet has outgrown it.  2022-05-05)

    if (!sails.config.custom.mergeFreezeAccessToken) {
      throw new Error('An access token for the MergeFreeze API (sails.config.custom.mergeFreezeAccessToken) is required to enable automated unfreezing/freezing of changes based on the files they change.  Please ask for help in #g-website, whether you are testing locally or using this as a live webhook.');
    }

    if (!sails.config.custom.slackWebhookUrlForGithubBot) {
      throw new Error('No Slack webhook URL configured for the GitHub bot to notify with alerts!  (Please set `sails.config.custom.slackWebhookUrlForGithubBot`.)');
    }//•

    if (!sails.config.custom.githubBotWebhookSecret) {
      throw new Error('No GitHub bot webhook secret configured!  (Please set `sails.config.custom.githubBotWebhookSecret`.)');
    }//•
    if (sails.config.custom.githubBotWebhookSecret !== botSignature) {
      throw new Error('Received unexpected GitHub webhook request with botSignature set to: '+botSignature);
    }//•

    if (!sails.config.custom.githubAccessToken) {
      throw new Error('No GitHub access token configured!  (Please set `sails.config.custom.githubAccessToken`.)');
    }//•

    if(!sails.config.custom.engMetricsGcpServiceAccountKey) {
      throw new Error('No GCP service account key configured!  (Please set `sails.config.custom.engMetricsGcpServiceAccountKey`.)');
    }//•

    let issueOrPr = (pr || issue || undefined);


    let ghNoun = this.req.get('X-GitHub-Event');// See https://developer.github.com/v3/activity/events/types/
    sails.log.verbose(`Received GitHub webhook request: ${ghNoun} :: ${action}: ${require('util').inspect({sender, repository: _.isObject(repository) ? repository.full_name : undefined, comment, label, issueOrPr}, {depth:null})}`);

    if (
      (ghNoun === 'issues' &&        ['opened','reopened'].includes(action))
    ) {
      //  ██╗███████╗███████╗██╗   ██╗███████╗
      //  ██║██╔════╝██╔════╝██║   ██║██╔════╝
      //  ██║███████╗███████╗██║   ██║█████╗
      //  ██║╚════██║╚════██║██║   ██║██╔══╝
      //  ██║███████║███████║╚██████╔╝███████╗
      //  ╚═╝╚══════╝╚══════╝ ╚═════╝ ╚══════╝
      //
      //   ██╗ ██████╗ ██████╗ ███████╗███╗   ██╗███████╗██████╗         ██╗    ██████╗ ███████╗ ██████╗ ██████╗ ███████╗███╗   ██╗███████╗██████╗ ██╗
      //  ██╔╝██╔═══██╗██╔══██╗██╔════╝████╗  ██║██╔════╝██╔══██╗       ██╔╝    ██╔══██╗██╔════╝██╔═══██╗██╔══██╗██╔════╝████╗  ██║██╔════╝██╔══██╗╚██╗
      //  ██║ ██║   ██║██████╔╝█████╗  ██╔██╗ ██║█████╗  ██║  ██║      ██╔╝     ██████╔╝█████╗  ██║   ██║██████╔╝█████╗  ██╔██╗ ██║█████╗  ██║  ██║ ██║
      //  ██║ ██║   ██║██╔═══╝ ██╔══╝  ██║╚██╗██║██╔══╝  ██║  ██║     ██╔╝      ██╔══██╗██╔══╝  ██║   ██║██╔═══╝ ██╔══╝  ██║╚██╗██║██╔══╝  ██║  ██║ ██║
      //  ╚██╗╚██████╔╝██║     ███████╗██║ ╚████║███████╗██████╔╝    ██╔╝       ██║  ██║███████╗╚██████╔╝██║     ███████╗██║ ╚████║███████╗██████╔╝██╔╝
      //   ╚═╝ ╚═════╝ ╚═╝     ╚══════╝╚═╝  ╚═══╝╚══════╝╚═════╝     ╚═╝        ╚═╝  ╚═╝╚══════╝ ╚═════╝ ╚═╝     ╚══════╝╚═╝  ╚═══╝╚══════╝╚═════╝ ╚═╝
      //
      // // Handle opened/reopened issue by commenting on it.
      // let owner = repository.owner.login;
      // let repo = repository.name;
      // let issueNumber = issueOrPr.number;
      // let newBotComment;
      // if (action === 'opened') {
      //   if (issueOrPr.state !== 'open') {
      //     newBotComment = '';// « checked below
      //   } else {
      //     newBotComment =
      //     `@${issueOrPr.user.login} Thanks for posting!  We'll take a look as soon as possible.\n`+
      //     `\n`+
      //     `In the mean time, there are a few ways you can help speed things along:\n`+
      //     ` - look for a workaround.  _(Even if it's just temporary, sharing your solution can save someone else a lot of time and effort.)_\n`+
      //     ` - tell us why this issue is important to you and your team.  What are you trying to accomplish?  _(Submissions with a little bit of human context tend to be easier to understand and faster to resolve.)_\n`+
      //     ` - make sure you've provided clear instructions on how to reproduce the bug from a clean install.\n`+
      //     ` - double-check that you've provided all of the requested version and dependency information.  _(Some of this info might seem irrelevant at first, like which database adapter you're using, but we ask that you include it anyway.  Oftentimes an issue is caused by a confluence of unexpected factors, and it can save everybody a ton of time to know all the details up front.)_\n`+
      //     ` - read the [code of conduct](https://sailsjs.com/documentation/contributing/code-of-conduct).\n`+
      //     ` - if appropriate, ask your business to [spons  or your issue](https://sailsjs.com/support).   _(Open source is our passion, and our core maintainers volunteer many of their nights and weekends working on Sails.  But you only get so many nights and weekends in life, and stuff gets done a lot faster when you can work on it during normal daylight hours.)_\n`+
      //     ` - let us know if you are using a 3rd party plugin; whether that's a database adapter, a non-standard view engine, or any other dependency maintained by someone other than our core team.  _(Besides the name of the 3rd party package, it helps to include the exact version you're using.  If you're unsure, check out [this list of all the core packages we maintain](https://sailsjs.com/architecture).)_ \n`+
      //     `<hr/>\n`+
      //     `\n`+
      //     `Please remember: never post in a public forum if you believe you've found a genuine security vulnerability.  Instead, [disclose it responsibly](https://sailsjs.com/security).\n`+
      //     `\n`+
      //     `For help with questions about Sails, [click here](http://sailsjs.com/support).\n`;
      //   }
      // } else {
      //   let wasReopenedByBot = GITHUB_USERNAMES_OF_BOTS_AND_MAINTAINERS.includes(sender.login.toLowerCase());
      //   if (wasReopenedByBot) {
      //     newBotComment = '';// « checked below
      //   } else {
      //     let greenLabels = _.filter(issueOrPr.labels, ({color}) => color === GREEN_LABEL_COLOR);
      //     await sails.helpers.flow.simultaneouslyForEach(greenLabels, async(greenLabel)=>{
      //       await sails.helpers.http.del('https://api.github.com/repos/'+encodeURIComponent(owner)+'/'+encodeURIComponent(repo)+'/issues/'+encodeURIComponent(issueNumber)+'/labels/'+encodeURIComponent(greenLabel.name),
      //         {},
      //         {
      //           'User-Agent': 'Fleetie pie',
      //           'Authorization': 'token '+sails.config.custom.githubAccessToken
      //         }
      //       );
      //     });//∞ß
      //     newBotComment =
      //     `Oh hey again, @${issueOrPr.user.login}.  Now that this issue is reopened, we'll take a fresh look as soon as we can!\n`+
      //     `<hr/>\n`+
      //     `\n`+
      //     `Please remember: never post in a public forum if you believe you've found a genuine security vulnerability.  Instead, [disclose it responsibly](https://sailsjs.com/security).\n`+
      //     `\n`+
      //     `For help with questions about Sails, see the [Sails support page](http://sailsjs.com/support).\n`;
      //   }
      // }
      // // Now that we know what to say, add our comment.
      // if (newBotComment) {
      //   await sails.helpers.http.post('https://api.github.com/repos/'+encodeURIComponent(owner)'/'+encodeURIComponent(repo)+'/issues/'+encodeURIComponent(issueNumber)+'/comments',
      //     {'body': newBotComment},
      //     {'Authorization': 'token '+sails.config.custom.githubAccessToken}
      //   );
      // }//ﬁ

    } else if (
      (ghNoun === 'issues' &&  ['closed'].includes(action))
    ) {
      //  ██╗███████╗███████╗██╗   ██╗███████╗     ██████╗██╗      ██████╗ ███████╗███████╗██████╗
      //  ██║██╔════╝██╔════╝██║   ██║██╔════╝    ██╔════╝██║     ██╔═══██╗██╔════╝██╔════╝██╔══██╗
      //  ██║███████╗███████╗██║   ██║█████╗      ██║     ██║     ██║   ██║███████╗█████╗  ██║  ██║
      //  ██║╚════██║╚════██║██║   ██║██╔══╝      ██║     ██║     ██║   ██║╚════██║██╔══╝  ██║  ██║
      //  ██║███████║███████║╚██████╔╝███████╗    ╚██████╗███████╗╚██████╔╝███████║███████╗██████╔╝
      //  ╚═╝╚══════╝╚══════╝ ╚═════╝ ╚══════╝     ╚═════╝╚══════╝ ╚═════╝ ╚══════╝╚══════╝╚═════╝
      //
      //
      // Handle closed issue by commenting on it.
      let owner = repository.owner.login;
      let repo = repository.name;
      let issueNumber = issueOrPr.number;
      let newBotComment;
      let baseHeadersForGithubApiRequests = {
        'User-Agent': 'Fleetie pie',
        'Authorization': `token ${sails.config.custom.githubAccessToken}`
      };

      if (!sails.config.custom.openAiSecret) {
        throw new Error('sails.config.custom.openAiSecret not set.  Cannot respond with haiku.');
      }//•

      // Generate haiku
      let BASE_MODEL = 'gpt-4';// The base model to use.  https://platform.openai.com/docs/models/gpt-4
      let MAX_TOKENS = 8000;// (Max tokens for gpt-3.5 ≈≈ 4000) (Max tokens for gpt-4 ≈≈ 8000)

      // Grab issue title and body, then truncate the length of the body so that it fits
      // within the maximum length tolerated by OpenAI.  Then combine those into a prompt
      // generate a haiku based on this issue.
      let issueSummary = '# ' + issueOrPr.title + '\n' + _.trunc(issueOrPr.body, MAX_TOKENS);

      // [?] API: https://platform.openai.com/docs/api-reference/chat/create
      let openAiReport = await sails.helpers.http.post('https://api.openai.com/v1/chat/completions', {
        model: BASE_MODEL,
        messages: [// https://platform.openai.com/docs/guides/chat/introduction
          {
            role: 'user',
            content: `You are an empathetic product designer.  I will give you a Github issue with information about a particular improvement to Fleet, an open-source device management and security platform.  You will write a haiku about how this improvement could benefit users or contributors.  Be detailed and specific in the haiku.  Do not use hyperbole.  Be matter-of-fact.  Be positive.  Do not make Fleet (or anyone) sound bad.  But be honest.  If appropriate, mention imagery from nature, or from a glass city in the clouds.  Do not give orders.\n\nThe first GitHub issue is:\n${issueSummary}`,
          }
        ],
        temperature: 0.7,
        max_tokens: 256//eslint-disable-line camelcase
      }, {
        Authorization: `Bearer ${sails.config.custom.openAiSecret}`
      })
      .tolerate((err)=>{
        sails.log('Failed to generate haiku using OpenAI.  Error details from OpenAI:',err);
      });

      if (!openAiReport) {// If OpenAI could not be reached…
        newBotComment = 'I couldn\'t think of a haiku this time.  (See fleetdm.com logs for more information.)';
      } else {// Otherwise, haiku was successfully generated…
        newBotComment = openAiReport.choices[0].message.content;
        newBotComment = newBotComment.replace(/^\s*\n*[^\n:]*Haiku[^\n:]*:\s*/i,'');// « eliminate "*Haiku:" prefix line, if one is generated
      }

      // Now that we know what to say, add our comment.
      await sails.helpers.http.post('https://api.github.com/repos/'+encodeURIComponent(owner)+'/'+encodeURIComponent(repo)+'/issues/'+encodeURIComponent(issueNumber)+'/comments',
        {'body': newBotComment},
        baseHeadersForGithubApiRequests
      ).tolerate((err)=>{
        sails.log.warn(`When the receive-from-github webhook sent a request to post a Haiku on a closed issue (${owner}/${repo} #${issueNumber}), an error occured. Full error: ${require('util').inspect(err)}`);
      });

    } else if (
      (ghNoun === 'pull_request' &&  ['opened','reopened','edited', 'synchronize', 'ready_for_review'].includes(action))
    ) {
      //  ██████╗ ██╗   ██╗██╗     ██╗         ██████╗ ███████╗ ██████╗ ██╗   ██╗███████╗███████╗████████╗
      //  ██╔══██╗██║   ██║██║     ██║         ██╔══██╗██╔════╝██╔═══██╗██║   ██║██╔════╝██╔════╝╚══██╔══╝
      //  ██████╔╝██║   ██║██║     ██║         ██████╔╝█████╗  ██║   ██║██║   ██║█████╗  ███████╗   ██║
      //  ██╔═══╝ ██║   ██║██║     ██║         ██╔══██╗██╔══╝  ██║▄▄ ██║██║   ██║██╔══╝  ╚════██║   ██║
      //  ██║     ╚██████╔╝███████╗███████╗    ██║  ██║███████╗╚██████╔╝╚██████╔╝███████╗███████║   ██║
      //  ╚═╝      ╚═════╝ ╚══════╝╚══════╝    ╚═╝  ╚═╝╚══════╝ ╚══▀▀═╝  ╚═════╝ ╚══════╝╚══════╝   ╚═╝
      //
      //   ██╗ ██████╗ ██████╗ ███████╗███╗   ██╗███████╗██████╗         ██╗    ███████╗██████╗ ██╗████████╗███████╗██████╗         ██╗    ██████╗ ███████╗ ██████╗ ██████╗ ███████╗███╗   ██╗███████╗██████╗ ██╗
      //  ██╔╝██╔═══██╗██╔══██╗██╔════╝████╗  ██║██╔════╝██╔══██╗       ██╔╝    ██╔════╝██╔══██╗██║╚══██╔══╝██╔════╝██╔══██╗       ██╔╝    ██╔══██╗██╔════╝██╔═══██╗██╔══██╗██╔════╝████╗  ██║██╔════╝██╔══██╗╚██╗
      //  ██║ ██║   ██║██████╔╝█████╗  ██╔██╗ ██║█████╗  ██║  ██║      ██╔╝     █████╗  ██║  ██║██║   ██║   █████╗  ██║  ██║      ██╔╝     ██████╔╝█████╗  ██║   ██║██████╔╝█████╗  ██╔██╗ ██║█████╗  ██║  ██║ ██║
      //  ██║ ██║   ██║██╔═══╝ ██╔══╝  ██║╚██╗██║██╔══╝  ██║  ██║     ██╔╝      ██╔══╝  ██║  ██║██║   ██║   ██╔══╝  ██║  ██║     ██╔╝      ██╔══██╗██╔══╝  ██║   ██║██╔═══╝ ██╔══╝  ██║╚██╗██║██╔══╝  ██║  ██║ ██║
      //  ╚██╗╚██████╔╝██║     ███████╗██║ ╚████║███████╗██████╔╝    ██╔╝       ███████╗██████╔╝██║   ██║   ███████╗██████╔╝    ██╔╝       ██║  ██║███████╗╚██████╔╝██║     ███████╗██║ ╚████║███████╗██████╔╝██╔╝
      //   ╚═╝ ╚═════╝ ╚═╝     ╚══════╝╚═╝  ╚═══╝╚══════╝╚═════╝     ╚═╝        ╚══════╝╚═════╝ ╚═╝   ╚═╝   ╚══════╝╚═════╝     ╚═╝        ╚═╝  ╚═╝╚══════╝ ╚═════╝ ╚═╝     ╚══════╝╚═╝  ╚═══╝╚══════╝╚═════╝ ╚═╝
      //

      let owner = repository.owner.login;
      let repo = repository.name;
      let prNumber = issueOrPr.number;

      // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
      // Want to do more?
      //
      // For some working, recent, easily-tweaked example code that manages a conversation with the GitHub bot
      // to get help submitters of PRs/issues get them up to spec, see:
      // https://github.com/fleetdm/fleet/blob/0a59adc2dd65bce5c1201a752e9c218faea7be35/website/api/controllers/webhooks/receive-from-github.js#L145-L216
      //
      // To potentially reuse:
      //     let newBotComment =
      //     `Oh hey again, @${issueOrPr.user.login}.  Now that this pull request is reopened, it's on our radar.  Please let us know if there's any new information we should be aware of!\n`+
      //     `<hr/>\n`+
      //     `\n`+
      //     `Please remember: never post in a public forum if you believe you've found a genuine security vulnerability.  Instead, [disclose it responsibly](https://sailsjs.com/security).\n`+
      //     `\n`+
      //     `For help with questions about Sails, see [Sails support](http://sailsjs.com/support).\n`;
      // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

      if (action === 'edited' && pr.state !== 'open') {// PR edited ‡
        // This is an edit to an already-closed pull request.
        // (Do nothing.)
      } else {// Either:
        // PR opened ‡  (Newly opened.)
        // PR reopened ‡   (This is a closed pull request, being reopened.  `action === 'reopened'`)

        let baseHeaders = {
          'User-Agent': 'Fleetie pie',
          'Authorization': `token ${sails.config.custom.githubAccessToken}`
        };

        require('assert')(sender.login !== undefined);

        //  ┌─┐┌─┐┌┬┐   ┬   ┌┬┐┌─┐┌┐┌┌─┐┌─┐┌─┐  ┌─┐─┐ ┬┌─┐┌─┐┌─┐┌┬┐┌─┐┌┬┐  ┬─┐┌─┐┬  ┬┬┌─┐┬ ┬┌─┐┬─┐┌─┐
        //  │ ┬├┤  │   ┌┼─  │││├─┤│││├─┤│ ┬├┤   ├┤ ┌┴┬┘├─┘├┤ │   │ ├┤  ││  ├┬┘├┤ └┐┌┘│├┤ │││├┤ ├┬┘└─┐
        //  └─┘└─┘ ┴   └┘   ┴ ┴┴ ┴┘└┘┴ ┴└─┘└─┘  └─┘┴ └─┴  └─┘└─┘ ┴ └─┘─┴┘  ┴└─└─┘ └┘ ┴└─┘└┴┘└─┘┴└─└─┘
        let DRI_BY_PATH = {};
        if (repo === 'fleet') {
          DRI_BY_PATH = sails.config.custom.githubRepoDRIByPath;
        } else {
          // Other repos don't have this configured.
          // FUTURE: Configure it for them
        }

        // Determine DRIs to request review from.
        //   > History: https://github.com/fleetdm/fleet/pull/12786)
        let expectedReviewers = [];//« GitHub usernames of people who we expect reviews from.

        // Look up already-requested reviewers
        // (for use later in minimizing extra notifications for editing PRs to contain new changes
        // while also still doing appropriate review requests.  Also for determining whether
        // to apply the ~ceo label)
        //
        // The "requested_reviewers" key in the pull request object:
        //   - https://developer.github.com/v3/activity/events/types
        //   - https://docs.github.com/en/webhooks-and-events/webhooks/webhook-events-and-payloads?actionType=edited#pull_request
        //   - https://docs.github.com/en/rest/pulls/pulls?apiVersion=2022-11-28#get-a-pull-request
        let alreadyRequestedReviewers = _.isArray(issueOrPr.requested_reviewers) ? _.pluck(issueOrPr.requested_reviewers, 'login') : [];
        alreadyRequestedReviewers = alreadyRequestedReviewers.map((username) => username.toLowerCase());// « make sure they are all lowercased

        // Look up paths
        // [?] https://docs.github.com/en/rest/reference/pulls#list-pull-requests-files
        let changedPaths = _.pluck(await sails.helpers.http.get(`https://api.github.com/repos/${owner}/${repo}/pulls/${prNumber}/files`, {
          per_page: 100,//eslint-disable-line camelcase
        }, baseHeaders).retry(), 'filename');// (don't worry, it's the whole path, not the filename)

        // Create an array of paths that will determine if the "~ga4-annotation" label will be automatically added to this PR.
        let CHANGED_PATHS_THAT_CREATE_ANALYTICS_ANNOTATIONS = [ 'website/views/pages/homepage.ejs', 'website/views/pages/pricing.ejs', 'website/views/partials/primary-tagline.partial.ejs'];
        let prShouldCreateGoogleAnalyticsAnnotation = false;

        // For each changed file, decide what reviewer to request, if any…
        for (let changedPath of changedPaths) {
          changedPath = changedPath.replace(/\/+$/,'');// « trim trailing slashes, just in case (b/c otherwise could loop forever)
          sails.log.verbose(`…checking DRI of changed path "${changedPath}"`);
          // If any of the changed paths are included in the CHANGED_PATHS_THAT_CREATE_ANALYTICS_ANNOTATIONS array, set the prShouldCreateGoogleAnalyticsAnnotation flag to true.
          if(CHANGED_PATHS_THAT_CREATE_ANALYTICS_ANNOTATIONS.includes(changedPath)) {
            prShouldCreateGoogleAnalyticsAnnotation = true;
          }
          let reviewer = undefined;//« whether to request review for this change
          let exactMatchDri = DRI_BY_PATH[changedPath];
          if (exactMatchDri) {// « If we've found our DRI, then we'll stop looking (for *this* changed path, anyway)
            reviewer = exactMatchDri;
          } else {// If there's no DRI for this *exact* file path, then check ancestral paths for the nearest DRI
            let numRemainingPathsToCheck = changedPath.split('/').length - 1;
            while (numRemainingPathsToCheck > 0) {
              let ancestralPath = changedPath.split('/').slice(0, numRemainingPathsToCheck).join('/');
              sails.log.verbose(`…checking DRI of ancestral path "${ancestralPath}" for changed path "${changedPath}"`);
              let nearestAncestralDri = DRI_BY_PATH[ancestralPath];// this is like the "catch-all" DRI, for a higher-level path
              if (nearestAncestralDri) {// Otherwise, if we have our DRI, we can stop here.
                reviewer = nearestAncestralDri;
                break;
              }//ﬁ
              numRemainingPathsToCheck--;
            }//∞
          }

          if (reviewer) {
            expectedReviewers.push(reviewer);
            expectedReviewers = _.uniq(expectedReviewers);// « avoid attempting to request review from the same person twice
          }//ﬁ

        }//∞

        // Now, if reviews should be requested for this PR, do so.
        //
        // > Note: Should we automatically remove reviewers?  Nah, we excluded this on purpose, to avoid removing deliberate
        // > custom review requests sent by real humans humans.
        if (!issueOrPr.draft) {// « (Draft PRs are skipped)
          let newReviewers;
          newReviewers = _.difference(expectedReviewers, alreadyRequestedReviewers);// « Don't request review from people whose review has already been requested.
          newReviewers = _.difference(newReviewers, [// « If the original PR author OR you, the sender (current PR author/editor) are the DRI, then don't request review.  No need to request review from yourself, and you CAN'T request review from the author (or the GitHub API will respond with an error.)
            issueOrPr.user.login.toLowerCase(),//« author (the original PR opener)  --  See `user.login` in https://docs.github.com/en/rest/pulls/pulls?apiVersion=2022-11-28#get-a-pull-request
            sender.login.toLowerCase(),//« sender (*you*, the current PR opener/editor)
          ]);
          if (newReviewers.length >= 1) {// « don't attempt to request review from no one
            // [?] https://docs.github.com/en/rest/pulls/review-requests?apiVersion=2022-11-28#request-reviewers-for-a-pull-request
            await sails.helpers.http.post(`https://api.github.com/repos/${owner}/${repo}/pulls/${prNumber}/requested_reviewers`, {
              reviewers: newReviewers,
            }, baseHeaders)
            .tolerate((err)=>{
              sails.log.warn(`When the receive-from-github webhook sent a request to add reviewers to an open pull request (${owner}/${repo} #${prNumber}), an error occured. Full error: ${require('util').inspect(err)}`);
            });
          }//ﬁ
        }//ﬁ

        //  ┌┬┐┌─┐┌┐┌┌─┐┌─┐┌─┐  ┬  ┌─┐┌┐ ┌─┐┬  ┌─┐
        //  │││├─┤│││├─┤│ ┬├┤   │  ├─┤├┴┐├┤ │  └─┐
        //  ┴ ┴┴ ┴┘└┘┴ ┴└─┘└─┘  ┴─┘┴ ┴└─┘└─┘┴─┘└─┘
        // Now manage automatic labeling.
        let existingLabels = _.isArray(issueOrPr.labels) ? _.pluck(issueOrPr.labels, 'name') : [];

        // Add the #handbook label to PRs that only make changes to the handbook,
        // and remove it from PRs that NO LONGER ONLY contain changes to the handbook.
        let isHandbookPR = false;
        if(repo === 'fleet'){
          isHandbookPR = await sails.helpers.githubAutomations.getIsPrOnlyHandbookChanges.with({prNumber: prNumber});
        }//ﬁ
        if(isHandbookPR && !existingLabels.includes('#handbook')) {
          // [?] https://docs.github.com/en/rest/issues/labels#add-labels-to-an-issue
          await sails.helpers.http.post(`https://api.github.com/repos/${owner}/${repo}/issues/${prNumber}/labels`, {
            labels: ['#handbook']
          }, baseHeaders);
        } else if (!isHandbookPR && existingLabels.includes('#handbook')) {
          // [?] https://docs.github.com/en/rest/issues/labels?apiVersion=2022-11-28#remove-a-label-from-an-issue
          await sails.helpers.http.del(`https://api.github.com/repos/${owner}/${repo}/issues/${prNumber}/labels/${encodeURIComponent('#handbook')}`, {}, baseHeaders)
          .tolerate({ exit: 'non200Response', raw: {statusCode: 404} }, (err)=>{// if the PR has gone missing, swallow the error and warn instead.
            sails.log.warn(`When trying to send a request to remove the #handbook label from PR #${prNumber} in the ${owner}/${repo} repo, an error occured. full error: ${require('util').inspect(err)}`);
          });
        }//ﬁ

        // Add the appropriate label to PRs awaiting review from the CEO so that these PRs show up in kanban.
        // [?] https://docs.github.com/en/webhooks-and-events/webhooks/webhook-events-and-payloads?actionType=edited#pull_request
        let isPRStillDependentOnAndReadyForCeoReview = expectedReviewers.includes('mikermcneil') && !issueOrPr.draft;
        if (isPRStillDependentOnAndReadyForCeoReview && !existingLabels.includes('~ceo')) {
          // [?] https://docs.github.com/en/rest/issues/labels#add-labels-to-an-issue
          await sails.helpers.http.post(`https://api.github.com/repos/${owner}/${repo}/issues/${prNumber}/labels`, {
            labels: ['~ceo']
          }, baseHeaders);
        } else if (!isPRStillDependentOnAndReadyForCeoReview && existingLabels.includes('~ceo')) {
          // [?] https://docs.github.com/en/rest/issues/labels?apiVersion=2022-11-28#remove-a-label-from-an-issue
          await sails.helpers.http.del(`https://api.github.com/repos/${owner}/${repo}/issues/${prNumber}/labels/${encodeURIComponent('~ceo')}`, {}, baseHeaders)
          .tolerate({ exit: 'non200Response', raw: {statusCode: 404} }, (err)=>{// if the PR has gone missing, swallow the error and warn instead.
            sails.log.warn(`When trying to send a request to remove the ~ceo label from PR #${prNumber} in the ${owner}/${repo} repo, an error occured. full error: ${require('util').inspect(err)}`);
          });
        }//ﬁ

        // If the prShouldCreateGoogleAnalyticsAnnotation was set to true, and this PR does not already have the ~ga4-annotation label, add it.
        // Note: unlike the #handbook and ~ceo labels, we don't automatically remove this label if it is added to a pull request, because it may have been added manually.
        if(prShouldCreateGoogleAnalyticsAnnotation && !existingLabels.includes('~ga4-annotation')) {
          // [?] https://docs.github.com/en/rest/issues/labels#add-labels-to-an-issue
          await sails.helpers.http.post(`https://api.github.com/repos/${owner}/${repo}/issues/${prNumber}/labels`, {
            labels: ['~ga4-annotation']
          }, baseHeaders).tolerate((err)=>{
            sails.log.warn(`When the receive-from-github webhook sent a request to add the "~ga4-annotation" label on an open pull request (${owner}/${repo} #${prNumber}), an error occured. Full error: ${require('util').inspect(err)}`);
          });
        }//ﬁ

        //  ┌─┐┬ ┬┌┬┐┌─┐   ┌─┐┌─┐┌─┐┬─┐┌─┐┬  ┬┌─┐   ┬   ┬ ┬┌┐┌┌─┐┬─┐┌─┐┌─┐┌─┐┌─┐
        //  ├─┤│ │ │ │ │───├─┤├─┘├─┘├┬┘│ │└┐┌┘├┤   ┌┼─  │ ││││├┤ ├┬┘├┤ ├┤ ┌─┘├┤
        //  ┴ ┴└─┘ ┴ └─┘   ┴ ┴┴  ┴  ┴└─└─┘ └┘ └─┘  └┘   └─┘┘└┘└  ┴└─└─┘└─┘└─┘└─┘
        // Now, if appropriate, auto-approve the change.
        let isAutoApprovalExpected = await sails.helpers.githubAutomations.getIsPrPreapproved.with({
          repo: repo,
          prNumber: prNumber,
          githubUserToCheck: sender.login,
          isGithubUserMaintainerOrDoesntMatter: GITHUB_USERNAMES_OF_BOTS_AND_MAINTAINERS.includes(sender.login.toLowerCase())
        });
        // (2024-07-09): The Mergefreeze requests and related code in this webhook are disabled/commented out because we are no longer freezing the main branch.
        // FUTURE: Remove mergefreeze related code and leave a comment with a link to an older version of this file
        // -----------------------------------------------------
        // Check whether the "main" branch is currently frozen (i.e. a feature freeze)
        // [?] https://docs.mergefreeze.com/web-api#get-freeze-status
        // let mergeFreezeMainBranchStatusReport = await sails.helpers.http.get('https://www.mergefreeze.com/api/branches/fleetdm/fleet/main', { access_token: sails.config.custom.mergeFreezeAccessToken }) //eslint-disable-line camelcase
        // .tolerate(['non200Response', 'requestFailed', {name: 'TimeoutError'}], (err)=>{
        //   // If the MergeFreeze API returns a non 200 response, log a warning and continue under the assumption that the main branch is not frozen.
        //   sails.log.warn('When sending a request to the MergeFreeze API to get the status of the main branch, MergeFreeze did not respond with a 2xx status code.  (Error details forthcoming in just a sec.)  First, how to remediate: If the main branch is frozen, it will need to be manually unfrozen before PR #'+prNumber+' can be merged. Raw underlying error from MergeFreeze: '+err.stack);
        //   return { frozen: false };
        // });
        // sails.log.verbose('#'+prNumber+' is under consideration...  The MergeFreeze API claims that it current main branch "frozen" status is:',mergeFreezeMainBranchStatusReport.frozen);
        // let isMainBranchFrozen = mergeFreezeMainBranchStatusReport.frozen;
        // let isMainBranchFrozen = false;
        // // If the "main" branch is not currently frozen and we still have PR numbers in our pocketOfPrNumbersUnfrozen array. Clear out the values in the platform record.
        // if(!isMainBranchFrozen && pocketOfPrNumbersUnfrozen.length > 0) {
        //   await Platform.updateOne({id: platformRecord.id}).set({currentUnfrozenGitHubPrNumbers: []});
        // }
        if (isAutoApprovalExpected) {
          // [?] https://docs.github.com/en/rest/reference/pulls#create-a-review-for-a-pull-request
          await sails.helpers.http.post(`https://api.github.com/repos/${owner}/${repo}/pulls/${prNumber}/reviews`, {
            event: 'APPROVE'
          }, baseHeaders)
          .retry()
          .tolerate((err)=>{
            return new Error(`When the receive-from-github webhook sent a request to approve a pull request (${owner}/${repo} #${prNumber}) an error occured. Full error: ${require('util').inspect(err)}`);
          });
        }//ﬁ
        //   // If "main" is explicitly frozen, then unfreeze this PR because it no longer contains
        //   // (or maybe never did contain) changes to freezeworthy files.
        //   // Note: We'll only do this if the PR is from the fleetdm/fleet repo.
        //   if (isMainBranchFrozen && repo === 'fleet') {

        //     pocketOfPrNumbersUnfrozen = _.union(pocketOfPrNumbersUnfrozen, [ prNumber ]);
        //     sails.log.verbose('#'+prNumber+' autoapproved, main branch is frozen...  prNumbers unfrozen:',pocketOfPrNumbersUnfrozen);

        //     // [?] See May 6th, 2022 changelog, which includes this code sample:
        //     // (https://www.mergefreeze.com/news)
        //     // (but as of July 26, 2022, I didn't see it documented here: https://docs.mergefreeze.com/web-api#post-freeze-status)
        //     // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
        //     // > You may now freeze or unfreeze a single PR (while maintaining the overall freeze) via the Web API.
        //     // ```
        //     // curl -d "frozen=true&user_name=Scooby Doo&unblocked_prs=[3]" -X POST https://www.mergefreeze.com/api/branches/mergefreeze/core/master/?access_token=[Your access token]
        //     // ```
        //     // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

        //     await sails.helpers.http.post(`https://www.mergefreeze.com/api/branches/fleetdm/fleet/main?access_token=${encodeURIComponent(sails.config.custom.mergeFreezeAccessToken)}`, {
        //       user_name: 'fleet-release',//eslint-disable-line camelcase
        //       unblocked_prs: pocketOfPrNumbersUnfrozen,//eslint-disable-line camelcase
        //     });
        //     // Update the Platform record to have the current unfrozen PR numbers
        //     await Platform.updateOne({id: platformRecord.id}).set({currentUnfrozenGitHubPrNumbers: pocketOfPrNumbersUnfrozen});
        //   }//ﬁ

        // } else {
        //   // If "main" is explicitly frozen, then freeze this PR because it now contains
        //   // (or maybe always did contain) changes to freezeworthy files.
        //   // Note: We'll only do this if the PR is from the fleetdm/fleet repo.
        //   if (isMainBranchFrozen && repo === 'fleet') {

        //     pocketOfPrNumbersUnfrozen = _.difference(pocketOfPrNumbersUnfrozen, [ prNumber ]);
        //     sails.log.verbose('#'+prNumber+' not autoapproved, main branch is frozen...  prNumbers unfrozen:',pocketOfPrNumbersUnfrozen);

        //     // [?] See explanation above.
        //     await sails.helpers.http.post(`https://www.mergefreeze.com/api/branches/fleetdm/fleet/main?access_token=${encodeURIComponent(sails.config.custom.mergeFreezeAccessToken)}`, {
        //       user_name: 'fleet-release',//eslint-disable-line camelcase
        //       unblocked_prs: pocketOfPrNumbersUnfrozen,//eslint-disable-line camelcase
        //     });
        //     // Update the Platform record to have the current unfrozen PR numbers
        //     await Platform.updateOne({id: platformRecord.id}).set({currentUnfrozenGitHubPrNumbers: pocketOfPrNumbersUnfrozen});
        //   }//ﬁ

        //   // Is this in use?
        //   // > For context on the history of this bit of code, which has gone been
        //   // > implemented a couple of different ways, and gone back and forth, check out:
        //   // > https://github.com/fleetdm/fleet/pull/5628#issuecomment-1196175485
        //   // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
        //   // if (IS_FROZEN) {
        //   //   // [?] https://docs.github.com/en/rest/reference/pulls#create-a-review-for-a-pull-request
        //   //   await sails.helpers.http.post(`https://api.github.com/repos/${owner}/${repo}/pulls/${prNumber}/reviews`, {
        //   //     event: 'REQUEST_CHANGES',
        //   //     body: 'The repository is currently frozen for an upcoming release.  \n> After the freeze has ended, please dismiss this review.  \n\nIn case of emergency, you can dismiss this review and merge now.'
        //   //   }, baseHeaders);
        //   // }//ﬁ
        //   // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
        // }

      }
    } else if (ghNoun === 'pull_request' && ['closed'].includes(action)) {
      //
      //   ██████╗██╗      ██████╗ ███████╗███████╗██████╗     ██████╗ ██████╗ ███████╗
      //  ██╔════╝██║     ██╔═══██╗██╔════╝██╔════╝██╔══██╗    ██╔══██╗██╔══██╗██╔════╝
      //  ██║     ██║     ██║   ██║███████╗█████╗  ██║  ██║    ██████╔╝██████╔╝███████╗
      //  ██║     ██║     ██║   ██║╚════██║██╔══╝  ██║  ██║    ██╔═══╝ ██╔══██╗╚════██║
      //  ╚██████╗███████╗╚██████╔╝███████║███████╗██████╔╝    ██║     ██║  ██║███████║
      //   ╚═════╝╚══════╝ ╚═════╝ ╚══════╝╚══════╝╚═════╝     ╚═╝     ╚═╝  ╚═╝╚══════╝
      //
      // Check the labels of merged PRs when they are closed.
      if(issueOrPr.merged) {
        let labelsWhenPrWasClosed = _.isArray(issueOrPr.labels) ? _.pluck(issueOrPr.labels, 'name') : [];
        // If the PR has the ~ga4-annotation label, send a POST request to a Zapier webhook.
        if(labelsWhenPrWasClosed.includes('~ga4-annotation')) {
          // Send a POST request to Zapier with the pull request
          await sails.helpers.http.post.with({
            url: 'https://hooks.zapier.com/hooks/catch/3627242/2x2uq4c/',
            data: {
              'pullRequest': issueOrPr,
              'webhookSecret': sails.config.custom.zapierSandboxWebhookSecret,
            }
          })
          .timeout(5000)
          .tolerate(['non200Response', 'requestFailed', {name: 'TimeoutError'}], (err)=>{
            // Note that Zapier responds with a 2xx status code even if something goes wrong, so just because this message is not logged doesn't mean everything is hunky dory.  More info: https://github.com/fleetdm/fleet/pull/6380#issuecomment-1204395762
            sails.log.warn(`When trying to send information about a merged pull request to Zapier, an error occured. Raw error: ${require('util').inspect(err)}`);
            return;
          });
        }

      }
    } else if (ghNoun === 'issue_comment' && ['created'].includes(action) && (issueOrPr&&issueOrPr.state === 'open')) {
      //   ██████╗ ██████╗ ███╗   ███╗███╗   ███╗███████╗███╗   ██╗████████╗
      //  ██╔════╝██╔═══██╗████╗ ████║████╗ ████║██╔════╝████╗  ██║╚══██╔══╝
      //  ██║     ██║   ██║██╔████╔██║██╔████╔██║█████╗  ██╔██╗ ██║   ██║
      //  ██║     ██║   ██║██║╚██╔╝██║██║╚██╔╝██║██╔══╝  ██║╚██╗██║   ██║
      //  ╚██████╗╚██████╔╝██║ ╚═╝ ██║██║ ╚═╝ ██║███████╗██║ ╚████║   ██║
      //   ╚═════╝ ╚═════╝ ╚═╝     ╚═╝╚═╝     ╚═╝╚══════╝╚═╝  ╚═══╝   ╚═╝
      //
      //   ██╗ ██████╗ ███╗   ██╗           ██████╗ ██████╗ ███████╗███╗   ██╗          ██████╗ ██████╗      ██████╗ ██████╗     ██╗███████╗███████╗██╗   ██╗███████╗██╗
      //  ██╔╝██╔═══██╗████╗  ██║    ▄ ██╗▄██╔═══██╗██╔══██╗██╔════╝████╗  ██║▄ ██╗▄    ██╔══██╗██╔══██╗    ██╔═══██╗██╔══██╗    ██║██╔════╝██╔════╝██║   ██║██╔════╝╚██╗
      //  ██║ ██║   ██║██╔██╗ ██║     ████╗██║   ██║██████╔╝█████╗  ██╔██╗ ██║ ████╗    ██████╔╝██████╔╝    ██║   ██║██████╔╝    ██║███████╗███████╗██║   ██║█████╗   ██║
      //  ██║ ██║   ██║██║╚██╗██║    ▀╚██╔▀██║   ██║██╔═══╝ ██╔══╝  ██║╚██╗██║▀╚██╔▀    ██╔═══╝ ██╔══██╗    ██║   ██║██╔══██╗    ██║╚════██║╚════██║██║   ██║██╔══╝   ██║
      //  ╚██╗╚██████╔╝██║ ╚████║      ╚═╝ ╚██████╔╝██║     ███████╗██║ ╚████║  ╚═╝     ██║     ██║  ██║    ╚██████╔╝██║  ██║    ██║███████║███████║╚██████╔╝███████╗██╔╝
      //   ╚═╝ ╚═════╝ ╚═╝  ╚═══╝           ╚═════╝ ╚═╝     ╚══════╝╚═╝  ╚═══╝          ╚═╝     ╚═╝  ╚═╝     ╚═════╝ ╚═╝  ╚═╝    ╚═╝╚══════╝╚══════╝ ╚═════╝ ╚══════╝╚═╝
      //
      // Handle newly-created comment by ungreening its parent issue/PR (if appropriate).
      let owner = repository.owner.login;
      let repo = repository.name;
      let issueNumber = issueOrPr.number;

      let wasPostedByBot = GITHUB_USERNAMES_OF_BOTS_AND_MAINTAINERS.includes(sender.login.toLowerCase());
      if (!wasPostedByBot) {
        let greenLabels = _.filter(issueOrPr.labels, ({color}) => color === GREEN_LABEL_COLOR);
        await sails.helpers.flow.simultaneouslyForEach(greenLabels, async(greenLabel)=>{
          await sails.helpers.http.del('https://api.github.com/repos/'+encodeURIComponent(owner)+'/'+encodeURIComponent(repo)+'/issues/'+encodeURIComponent(issueNumber)+'/labels/'+encodeURIComponent(greenLabel.name),
            {},
            {
              'User-Agent': 'Fleetie Pie',
              'Authorization': 'token '+sails.config.custom.githubAccessToken
            }
          );
        });//∞ß
      }//ﬁ
    } else if (
      (ghNoun === 'issue_comment' && ['deleted'].includes(action) && !GITHUB_USERNAMES_OF_BOTS_AND_MAINTAINERS.includes(comment.user.login.toLowerCase()))||
      (ghNoun === 'commit_comment' && ['created'].includes(action) && !GITHUB_USERNAMES_OF_BOTS_AND_MAINTAINERS.includes(comment.user.login.toLowerCase()))||
      (ghNoun === 'label' && false /* label change notifications temporarily disabled until digital experience team has time to clean up labels.  FUTURE: turn this back on after doing that cleanup to facilitate gradual ongoing maintenance and education rather than herculean cleanup efforts and retraining */ && ['created','edited','deleted'].includes(action) && GITHUB_USERNAME_OF_DRI_FOR_LABELS !== sender.login.toLowerCase())||//« exempt label changes made by the directly responsible individual for labels, because otherwise when process changes/fiddlings happen, they can otherwise end up making too much noise in Slack
      (ghNoun === 'issue_comment' && ['created'].includes(action) && issueOrPr.state !== 'open' && (issueOrPr.closed_at) && ((new Date(issueOrPr.closed_at)).getTime() < Date.now() - 7*24*60*60*1000 ) && !GITHUB_USERNAMES_OF_BOTS_AND_MAINTAINERS.includes(sender.login.toLowerCase()) )
    ) {
      //  ██╗███╗   ██╗███████╗ ██████╗ ██████╗ ███╗   ███╗    ██╗   ██╗███████╗
      //  ██║████╗  ██║██╔════╝██╔═══██╗██╔══██╗████╗ ████║    ██║   ██║██╔════╝
      //  ██║██╔██╗ ██║█████╗  ██║   ██║██████╔╝██╔████╔██║    ██║   ██║███████╗
      //  ██║██║╚██╗██║██╔══╝  ██║   ██║██╔══██╗██║╚██╔╝██║    ██║   ██║╚════██║
      //  ██║██║ ╚████║██║     ╚██████╔╝██║  ██║██║ ╚═╝ ██║    ╚██████╔╝███████║
      //  ╚═╝╚═╝  ╚═══╝╚═╝      ╚═════╝ ╚═╝  ╚═╝╚═╝     ╚═╝     ╚═════╝ ╚══════╝
      //
      //   ██╗██╗███╗   ██╗    ███████╗██╗      █████╗  ██████╗██╗  ██╗██╗
      //  ██╔╝██║████╗  ██║    ██╔════╝██║     ██╔══██╗██╔════╝██║ ██╔╝╚██╗
      //  ██║ ██║██╔██╗ ██║    ███████╗██║     ███████║██║     █████╔╝  ██║
      //  ██║ ██║██║╚██╗██║    ╚════██║██║     ██╔══██║██║     ██╔═██╗  ██║
      //  ╚██╗██║██║ ╚████║    ███████║███████╗██║  ██║╚██████╗██║  ██╗██╔╝
      //   ╚═╝╚═╝╚═╝  ╚═══╝    ╚══════╝╚══════╝╚═╝  ╚═╝ ╚═════╝╚═╝  ╚═╝╚═╝
      //
      // Handle deleted issue/PR comments, new/modified/deleted commit comments,
      // new/edited/deleted labels, and new comments on closed issues/PRs by
      // posting to the Fleet Slack.
      // > FUTURE: also post to Slack about deleted issues, new repos, and deleted repos
      await sails.helpers.http.post(
        sails.config.custom.slackWebhookUrlForGithubBot,//« #g-marketing channel (Fleet Slack workspace)
        {
          text:
          (
            (ghNoun === 'issue_comment' && action === 'deleted') ?
              `@${sender.login} just deleted a GitHub comment that was originally posted at ${(new Date(comment.created_at)).toString()} by @${comment.user.login} in ${issueOrPr.html_url}.\n\nFormerly, the comment read:\n\`\`\`\n${comment.body}\n\`\`\``
            : (ghNoun === 'commit_comment') ?
              `@${sender.login} just created a new GitHub commit comment in ${repository.owner.login}/${repository.name}:\n\n> ${comment.html_url}\n\`\`\`\n${comment.body}\n\`\`\``
            : (ghNoun === 'label' && action === 'edited') ?
              `@${sender.login} just edited a GitHub label "*${label.name}*" (#${label.color}) in ${repository.owner.login}/${repository.name}.\n\nChanges:\n\`\`\`\n${Object.keys(changes).length === 0 ? 'GitHub did not report any changes.  This usually means the label description was updated (because label descriptions are not available via the GitHub API.)' : require('util').inspect(changes,{depth:null})}\n\`\`\`\n\n> To manage labels in ${repository.owner.login}/${repository.name}, visit https://github.com/${encodeURIComponent(repository.owner.login)}/${encodeURIComponent(repository.name)}/labels`
            : (ghNoun === 'label') ?
              `@${sender.login} just ${action} a GitHub label "*${label.name}*" (#${label.color}) in ${repository.owner.login}/${repository.name}.\n\n> To manage labels in ${repository.owner.login}/${repository.name}, visit https://github.com/${encodeURIComponent(repository.owner.login)}/${encodeURIComponent(repository.name)}/labels`
            :
              `@${sender.login} just created a zombie comment in a GitHub issue or PR that had already been closed for >7 days (${issueOrPr.html_url}):\n\n> ${comment.html_url}\n\`\`\`\n${comment.body}\n\`\`\``
          )+`\n`
        },
        {'Content-Type':'application/json'}
      )
      .timeout(5000)
      .retry([{name: 'TimeoutError'}, 'non200Response', 'requestFailed']);
    } else if(ghNoun === 'release' && ['published'].includes(action) ) {
      //  ██████╗ ███████╗██╗     ███████╗ █████╗ ███████╗███████╗███████╗
      //  ██╔══██╗██╔════╝██║     ██╔════╝██╔══██╗██╔════╝██╔════╝██╔════╝
      //  ██████╔╝█████╗  ██║     █████╗  ███████║███████╗█████╗  ███████╗
      //  ██╔══██╗██╔══╝  ██║     ██╔══╝  ██╔══██║╚════██║██╔══╝  ╚════██║
      //  ██║  ██║███████╗███████╗███████╗██║  ██║███████║███████╗███████║
      //  ╚═╝  ╚═╝╚══════╝╚══════╝╚══════╝╚═╝  ╚═╝╚══════╝╚══════╝╚══════╝
      //
      // Handle new Fleet releases by sending a POST request to Zapier to
      // trigger an automation that updates Slack channel topics with the latest version of Fleet.
      let owner = repository.owner.login;
      let repo = repository.name;

      // Only continue if this release came from the fleetdm/fleet repo,
      if(owner === 'fleetdm' && repo === 'fleet') {
        if(release
          && _.startsWith(release.tag_name, 'fleet-v')// Only send requests for releases with tag names that start with 'fleet'
          && _.endsWith(release.tag_name, '.0')// Only send requests if the release is a major or minor version. This works because all Fleet semvers have 2 periods.
        ) {
          // Send a POST request to Zapier with the release object.
          await sails.helpers.http.post.with({
            url: 'https://hooks.zapier.com/hooks/catch/3627242/3ozw6bk/',
            data: {
              'release': release,
              'webhookSecret': sails.config.custom.zapierSandboxWebhookSecret,
            }
          })
          .timeout(5000)
          .tolerate(['non200Response', 'requestFailed', {name: 'TimeoutError'}], (err)=>{
            // Note that Zapier responds with a 2xx status code even if something goes wrong, so just because this message is not logged doesn't mean everything is hunky dory.  More info: https://github.com/fleetdm/fleet/pull/6380#issuecomment-1204395762
            sails.log.warn(`When trying to send information about a new Fleet release to Zapier, an error occured. Raw error: ${require('util').inspect(err)}`);
            return;
          });
        }
      }//ﬁ
    } else if(ghNoun === 'projects_v2_item') {
      //
      //  ██████╗ ██████╗  ██████╗      ██╗███████╗ ██████╗████████╗███████╗    ██╗   ██╗██████╗
      //  ██╔══██╗██╔══██╗██╔═══██╗     ██║██╔════╝██╔════╝╚══██╔══╝██╔════╝    ██║   ██║╚════██╗
      //  ██████╔╝██████╔╝██║   ██║     ██║█████╗  ██║        ██║   ███████╗    ██║   ██║ █████╔╝
      //  ██╔═══╝ ██╔══██╗██║   ██║██   ██║██╔══╝  ██║        ██║   ╚════██║    ╚██╗ ██╔╝██╔═══╝
      //  ██║     ██║  ██║╚██████╔╝╚█████╔╝███████╗╚██████╗   ██║   ███████║     ╚████╔╝ ███████╗
      //  ╚═╝     ╚═╝  ╚═╝ ╚═════╝  ╚════╝ ╚══════╝ ╚═════╝   ╚═╝   ╚══════╝      ╚═══╝  ╚══════╝
      //
      /**
       * GitHub Projects v2 status change tracking
       *
       * This webhook event handler tracks issue status changes in GitHub Projects v2 for engineering metrics.
       *
       * Tracked projects:
       * - Orchestration
       * - MDM
       * - Software
       * - Security & compliance
       *
       * Status transitions tracked:
       *
       * 1. "In progress" status:
       *    - Triggers when: Status changes TO "in progress"
       *    - From states: "ready" or null (first time) OR any other state (if not already tracked)
       *    - Saves to: github_metrics.issue_status_change
       *    - Data: timestamp, repo, issue_number, status = 'in_progress'
       *
       * 2. "Awaiting QA" status:
       *    - Triggers when: Status changes TO "awaiting qa"
       *    - From states:
       *      - "in progress" or "review" → Always creates new row
       *      - Any other state → Creates row only if no QA ready row exists
       *    - Saves to: github_metrics.issue_qa_ready
       *    - Data: qa_ready time, assignee, issue details, time from in_progress to qa_ready
       *    - Note: Requires an existing in_progress record to calculate time
       *
       * 3. "Release" status:
       *    - Triggers when: Status changes TO "release"
       *    - From states:
       *      - "awaiting qa" → Always creates new row
       *      - Any other state → Creates row only if issue has been in QA (exists in issue_qa_ready)
       *    - Saves to: github_metrics.issue_release_ready
       *    - Data: release_ready time, assignee, issue details, time from in_progress to release_ready
       *    - Note: Requires an existing in_progress record to calculate time
       *
       * Time calculations:
       * - All time calculations can optionally exclude weekends (controlled by excludeWeekends flag)
       * - Weekend exclusion adjusts start/end times and subtracts weekend days from duration
       * - Times are calculated from the webhook's updated_at timestamp for accuracy
       *
       * Issue type classification:
       * - Based on GitHub issue labels:
       *   - "bug" → type: "bug"
       *   - "story" → type: "story"
       *   - "~sub-task" → type: "sub-task"
       *   - Otherwise → type: "other"
       */

      // Check and parse GCP service account key
      let gcpServiceAccountKey = await sails.helpers.flow.build(()=>{
        let parsedKey;

        // Check if it's already an object or needs parsing
        if (typeof sails.config.custom.engMetricsGcpServiceAccountKey === 'object') {
          parsedKey = sails.config.custom.engMetricsGcpServiceAccountKey;
        } else if (typeof sails.config.custom.engMetricsGcpServiceAccountKey === 'string') {
          // Fix common JSON formatting issues before parsing
          let jsonString = sails.config.custom.engMetricsGcpServiceAccountKey;

          // This handles cases where the private key has literal newlines
          jsonString = jsonString.replace(/"private_key":\s*"([^"]+)"/g, (match, key) => {
            // Replace actual newlines with escaped newlines only within the private key value
            const fixedKey = key.replace(/\n/g, '\\n');
            return `"private_key": "${fixedKey}"`;
          });

          // Parse the cleaned JSON
          parsedKey = JSON.parse(jsonString);
        } else {
          throw new Error('Invalid GCP service account key type');
        }

        // Validate that it has the expected structure
        if (!parsedKey.type || !parsedKey.project_id || !parsedKey.private_key) {
          throw new Error('Invalid GCP service account key structure');
        }

        return parsedKey;
      }).intercept((err)=>{
        return new Error(`An error occured when parsing the set 'sails.config.custom.engMetricsGcpServiceAccountKey' value. `, err);
      });


      // Process the status change inline
      let statusChange = null;

      // Check if this is a project item update with status change
      if (!changes || !changes.field_value) {
        // Not a status change we care about
        return;
      }

      const fieldValue = changes.field_value;
      // Check if this is a status field change
      if (fieldValue.field_name !== 'Status') {
        // Not a status field change
        return;
      }

      // Check if this is one of our tracked projects
      const projectNumber = fieldValue.project_number;
      const validProjects = Object.values(sails.config.custom.githubProjectsV2.projects);

      if (!validProjects.includes(projectNumber)) {
        sails.log.verbose(`Ignoring status change for project ${projectNumber} - not a tracked project`);
        return;
      }

      // Check if status changed to "in progress" from "ready" or null
      // from and to are either objects with a name property or null
      const fromStatus = fieldValue.from ? fieldValue.from.name.toLowerCase() : '';
      const toStatus = fieldValue.to ? fieldValue.to.name.toLowerCase() : '';

      // Log the status change for debugging
      sails.log.verbose(`Status change detected: "${fromStatus || '(null)'}" -> "${toStatus}"`);

      // Check if the "to" status includes "in progress", "awaiting qa", or "release"
      const isToInProgress = toStatus.includes('in progress');
      const isToAwaitingQa = toStatus.includes('awaiting qa');
      const isToRelease = toStatus.includes('release');

      if (!isToInProgress && !isToAwaitingQa && !isToRelease) {
        sails.log.verbose(`Ignoring status change - "to" status doesn't include "in progress", "awaiting qa", or "release": ${toStatus}`);
        return;
      }

      // Get issue details from the payload
      if (!projectsV2Item || !projectsV2Item.content_node_id) {
        sails.log.error('Missing projects_v2_item or content_node_id in payload');
        return;
      }

      // Fetch issue details from GitHub API
      // GitHub GraphQL API query to get issue details from node ID
      const queryToFindThisIssueOnGithub = `
        query($nodeId: ID!) {
          node(id: $nodeId) {
            ... on Issue {
              number
              repository {
                nameWithOwner
              }
              assignees(first: 1) {
                nodes {
                  login
                }
              }
              labels(first: 20) {
                nodes {
                  name
                }
              }
            }
          }
        }
      `;

      const graphqlQueryResponse = await sails.helpers.http.post('https://api.github.com/graphql',
        {
          query: queryToFindThisIssueOnGithub,
          variables: { nodeId: projectsV2Item.content_node_id }
        },
        {
          'Authorization': `Bearer ${sails.config.custom.githubAccessToken}`,
          'Accept': 'application/vnd.github.v4+json',
          'User-Agent': 'Fleet-Engineering-Metrics'
        }
      );

      if (!graphqlQueryResponse.data || !graphqlQueryResponse.data.node) {
        return;
      }

      const node = graphqlQueryResponse.data.node;
      const assignee = node.assignees.nodes.length > 0 ? node.assignees.nodes[0].login : '';

      // Extract label names
      const labels = node.labels.nodes.map(label => label.name.toLowerCase());

      // Determine issue type based on labels
      let issueType = 'other';
      if (labels.includes('bug')) {
        issueType = 'bug';
      } else if (labels.includes('story')) {
        issueType = 'story';
      } else if (labels.includes('~sub-task')) {
        issueType = 'sub-task';
      }

      let issueDetails = {
        repo: node.repository.nameWithOwner,
        issueNumber: node.number,
        assignee: assignee,
        type: issueType
      };

      // Handle "in progress" status changes
      if (isToInProgress) {
        // Check if the "from" status is null or includes "ready"
        const isFromNullOrReady = fieldValue.from === null || fromStatus.includes('ready');

        if (!isFromNullOrReady) {
          sails.log.verbose(`Status change from "${fromStatus}" to "in progress" - will check if already tracked`);
          const exists = await sails.helpers.engineeringMetrics.checkIfRecordExists.with({
            repo: issueDetails.repo,
            issueNumber: issueDetails.issueNumber,
            gcpServiceAccountKey: gcpServiceAccountKey,
            tableId: 'issue_status_change',
            additionalCondition: 'AND status = \'in_progress\''
          });
          if (exists) {
            sails.log.verbose(`Issue ${issueDetails.repo}#${issueDetails.issueNumber} already tracked as in_progress, skipping`);
            return;
          }
          sails.log.info(`Issue ${issueDetails.repo}#${issueDetails.issueNumber} not yet tracked, will save as in_progress`);
          // Prepare data for BigQuery
          const statusChangeData = {
            date: projectsV2Item.updated_at,  // Use the actual update time from webhook
            repo: issueDetails.repo,
            issue_number: issueDetails.issueNumber,  // eslint-disable-line camelcase
            status: 'in_progress'
          };

          // Save to BigQuery
          await sails.helpers.engineeringMetrics.saveToBigquery.with({
            data: statusChangeData,
            gcpServiceAccountKey: gcpServiceAccountKey,
            tableId: 'issue_status_change'
          });
          statusChange = statusChangeData;
        } else {
          // Prepare data for BigQuery
          const statusChangeData = {
            date: projectsV2Item.updated_at,  // Use the actual update time from webhook
            repo: issueDetails.repo,
            issue_number: issueDetails.issueNumber,  // eslint-disable-line camelcase
            status: 'in_progress'
          };

          // Save to BigQuery
          await sails.helpers.engineeringMetrics.saveToBigquery.with({
            data: statusChangeData,
            gcpServiceAccountKey: gcpServiceAccountKey,
            tableId: 'issue_status_change'
          });
          statusChange = statusChangeData;
        }
      }//ﬁ

      // Handle "awaiting qa" status changes
      if (isToAwaitingQa) {
        // Check if from status is "in progress" or "review"
        const isFromInProgressOrReview = fromStatus.includes('in progress') || fromStatus.includes('review');

        // Check if we should create a new QA ready row
        let shouldCreateQaRow = false;

        if (isFromInProgressOrReview) {
          // Always create if transitioning from in progress or review
          shouldCreateQaRow = true;
        } else {
          // Check if row already exists
          const qaRowExists = await sails.helpers.engineeringMetrics.checkIfRecordExists.with({
            repo: issueDetails.repo,
            issueNumber: issueDetails.issueNumber,
            gcpServiceAccountKey: gcpServiceAccountKey,
            tableId: 'issue_qa_ready'
          });
          if (!qaRowExists) {
            shouldCreateQaRow = true;
          } else {
            sails.log.verbose(`QA ready row already exists for ${issueDetails.repo}#${issueDetails.issueNumber}, skipping`);
          }
        }

        if (shouldCreateQaRow) {
          // Get the latest in_progress status from BigQuery
          let inProgressData = await sails.helpers.flow.build(async ()=>{
            const {BigQuery} = require('@google-cloud/bigquery');
            const bigquery = new BigQuery({
              projectId: gcpServiceAccountKey.project_id,
              credentials: gcpServiceAccountKey
            });
            // Configure dataset and table names
            const datasetId = 'github_metrics';
            const tableId = 'issue_status_change';

            // Query to get the latest in_progress status
            const query = `
              SELECT date, repo, issue_number
              FROM \`${gcpServiceAccountKey.project_id}.${datasetId}.${tableId}\`
              WHERE repo = @repo
                AND issue_number = @issueNumber
                AND status = 'in_progress'
              ORDER BY date DESC
              LIMIT 1
            `;

            const options = {
              query: query,
              params: {
                repo: issueDetails.repo,
                issueNumber: issueDetails.issueNumber
              }
            };

            // Run the query
            const [rows] = await bigquery.query(options);

            if (rows.length === 0) {
              return null;
            }

            // Convert BigQueryTimestamp to string if needed
            const result = rows[0];
            if (result.date && result.date.value) {
              result.date = result.date.value;
            }

            return result;
          }).tolerate((err)=>{
            // Handle specific BigQuery errors
            if (err.name === 'PartialFailureError') {
              // Log the specific rows that failed
              sails.log.warn(`When an issue (${issueDetails.repo}#${issueDetails.issueNumber}) was moved into the "Awaiting QA" column, there was a partial failure when getting latest in_progress status from BigQuery:`, err.errors);
            } else if (err.code === 404) {
              sails.log.warn(`When an issue (${issueDetails.repo}#${issueDetails.issueNumber}) was moved into the "Awaiting QA" column, in progress data could not be found. BigQuery table or dataset not found. Please ensure the table exists:`, {
                dataset: 'github_metrics',
                table: 'issue_status_change',
                fullError: err.message
              });
            } else if (err.code === 403) {
              sails.log.warn(`When an issue (${issueDetails.repo}#${issueDetails.issueNumber}) was moved into the "Awaiting QA" column, in progress data could not be found. Permission denied when accessing BigQuery. Check service account permissions.`);
            } else {
              sails.log.warn(`When an issue (${issueDetails.repo}#${issueDetails.issueNumber}) was moved into the "Awaiting QA" column, There was an error getting latest in_progress status from BigQuery:`, err);
            }
            return null;
          });

          if (inProgressData) {
            // Calculate time to QA ready
            const qaReadyTime = new Date(projectsV2Item.updated_at);  // Use webhook timestamp
            const inProgressTime = new Date(inProgressData.date);

            let timeToQaReadySeconds = await sails.helpers.flow.build(async ()=>{
              if (!sails.config.custom.githubProjectsV2.excludeWeekends) {
                // If weekend exclusion is disabled, return simple time difference
                return Math.floor((qaReadyTime - inProgressTime) / 1000);
              }

              // Use the provided weekend exclusion logic
              const startDay = inProgressTime.getUTCDay();
              const endDay = qaReadyTime.getUTCDay();

              // Case: Both start time and end time are on the same weekend
              if (
                (startDay === 0 || startDay === 6) &&
                (endDay === 0 || endDay === 6) &&
                Math.floor(qaReadyTime / (24 * 60 * 60 * 1000)) -
                Math.floor(inProgressTime / (24 * 60 * 60 * 1000)) <=
                2
              ) {
                // Return 0 seconds
                return 0;
              }

              // Make copies to avoid modifying original dates
              const adjustedStartTime = new Date(inProgressTime);
              const adjustedEndTime = new Date(qaReadyTime);

              // Set to start of Monday if start time is on weekend
              if (startDay === 0) {
                // Sunday
                adjustedStartTime.setUTCDate(adjustedStartTime.getUTCDate() + 1);
                adjustedStartTime.setUTCHours(0, 0, 0, 0);
              } else if (startDay === 6) {
                // Saturday
                adjustedStartTime.setUTCDate(adjustedStartTime.getUTCDate() + 2);
                adjustedStartTime.setUTCHours(0, 0, 0, 0);
              }

              // Set to start of Saturday if end time is on Sunday
              if (endDay === 0) {
                // Sunday
                adjustedEndTime.setUTCDate(adjustedEndTime.getUTCDate() - 1);
                adjustedEndTime.setUTCHours(0, 0, 0, 0);
              } else if (endDay === 6) {
                // Saturday
                adjustedEndTime.setUTCHours(0, 0, 0, 0);
              }

              // Count weekend days between adjusted dates
              // Make local copies for weekend counting
              let weekendStartDate = new Date(adjustedStartTime);
              let weekendEndDate = new Date(adjustedEndTime);

              // Ensure weekendStartDate is before weekendEndDate
              if (weekendStartDate > weekendEndDate) {
                [weekendStartDate, weekendEndDate] = [weekendEndDate, weekendStartDate];
              }

              // Make sure start dates and end dates are not on weekends. We just want to count the weekend days between them.
              if (weekendStartDate.getUTCDay() === 0) {
                weekendStartDate.setUTCDate(weekendStartDate.getUTCDate() + 1);
              } else if (weekendStartDate.getUTCDay() === 6) {
                weekendStartDate.setUTCDate(weekendStartDate.getUTCDate() + 2);
              }
              if (weekendEndDate.getUTCDay() === 0) {
                weekendEndDate.setUTCDate(weekendEndDate.getUTCDate() - 2);
              } else if (weekendEndDate.getUTCDay() === 6) {
                weekendEndDate.setUTCDate(weekendEndDate.getUTCDate() - 1);
              }

              let weekendDays = 0;
              const current = new Date(weekendStartDate);

              while (current <= weekendEndDate) {
                const day = current.getUTCDay();
                if (day === 0 || day === 6) {
                  // Sunday (0) or Saturday (6)
                  weekendDays++;
                }
                current.setUTCDate(current.getUTCDate() + 1);
              }

              // Calculate raw time difference in milliseconds
              const diffMs = adjustedEndTime - adjustedStartTime - weekendDays * 24 * 60 * 60 * 1000;

              // Ensure we don't return negative values
              return Math.max(0, Math.floor(diffMs / 1000));
            });

            // Determine project name by reverse lookup
            const projectName = Object.keys(sails.config.custom.githubProjectsV2.projects).find(
              key => sails.config.custom.githubProjectsV2.projects[key] === projectNumber
            ) || '';

            // Prepare QA ready data
            const qaReadyData = {
              qa_ready: qaReadyTime.toISOString().split('T')[0],  // eslint-disable-line camelcase
              assignee: issueDetails.assignee || '',  // Get assignee from issue details
              issue_url: `https://github.com/${issueDetails.repo}/issues/${issueDetails.issueNumber}`,  // eslint-disable-line camelcase
              time_to_qa_ready_seconds: timeToQaReadySeconds,  // eslint-disable-line camelcase
              repo: issueDetails.repo,
              issue_number: issueDetails.issueNumber,  // eslint-disable-line camelcase
              qa_ready_time: qaReadyTime.toISOString(),  // eslint-disable-line camelcase
              in_progress_time: inProgressTime.toISOString(),  // eslint-disable-line camelcase
              project: projectName,
              type: issueDetails.type  // Issue type based on labels
            };

            // Save to BigQuery
            await sails.helpers.engineeringMetrics.saveToBigquery.with({
              data: qaReadyData,
              gcpServiceAccountKey: gcpServiceAccountKey,
              tableId: 'issue_qa_ready'
            });

            sails.log.info('Saved QA ready metrics:', {
              repo: issueDetails.repo,
              issueNumber: issueDetails.issueNumber,
              timeToQaReadySeconds,
              project: projectName
            });

            statusChange = qaReadyData;
          } else {
            sails.log.info(`No in_progress status found for ${issueDetails.repo}#${issueDetails.issueNumber}, cannot calculate QA ready time`);
          }
        }
      }//ﬁ

      // Handle "release" status changes
      if (isToRelease) {
        // Check if from status is "awaiting qa"
        const isFromAwaitingQa = fromStatus.includes('awaiting qa');

        // Check if we should save this release transition
        let shouldSaveRelease = false;
        if (!isFromAwaitingQa) {
          // Not directly from "awaiting qa", check if issue has ever been in QA
          const hasBeenInQa = await sails.helpers.engineeringMetrics.checkIfRecordExists.with({
            repo: issueDetails.repo,
            issueNumber: issueDetails.issueNumber,
            gcpServiceAccountKey: gcpServiceAccountKey,
            tableId: 'issue_qa_ready'
          });
          if (hasBeenInQa) {
            sails.log.info(`Issue ${issueDetails.repo}#${issueDetails.issueNumber} transitioning to release (previously was in QA)`);
            shouldSaveRelease = true;
          } else {
            sails.log.verbose(`Issue ${issueDetails.repo}#${issueDetails.issueNumber} has never been in "awaiting qa", skipping release tracking`);
          }
        } else {
          shouldSaveRelease = true;
        }

        if (shouldSaveRelease) {
          // Get the latest in_progress status from BigQuery
          let inProgressData = await sails.helpers.flow.build(async ()=>{
            const {BigQuery} = require('@google-cloud/bigquery');
            const bigquery = new BigQuery({
              projectId: gcpServiceAccountKey.project_id,
              credentials: gcpServiceAccountKey
            });
            // Configure dataset and table names
            const datasetId = 'github_metrics';
            const tableId = 'issue_status_change';

            // Query to get the latest in_progress status
            const query = `
              SELECT date, repo, issue_number
              FROM \`${gcpServiceAccountKey.project_id}.${datasetId}.${tableId}\`
              WHERE repo = @repo
                AND issue_number = @issueNumber
                AND status = 'in_progress'
              ORDER BY date DESC
              LIMIT 1
            `;

            const options = {
              query: query,
              params: {
                repo: issueDetails.repo,
                issueNumber: issueDetails.issueNumber
              }
            };

            // Run the query
            const [rows] = await bigquery.query(options);

            if (rows.length === 0) {
              return null;
            }

            // Convert BigQueryTimestamp to string if needed
            const result = rows[0];
            if (result.date && result.date.value) {
              result.date = result.date.value;
            }

            return result;
          }).tolerate((err)=>{
            // Handle specific BigQuery errors
            if (err.name === 'PartialFailureError') {
              // Log the specific rows that failed
              sails.log.warn(`When an issue (${issueDetails.repo}#${issueDetails.issueNumber}) was moved into the "Ready for release" column, there was a partial failure when getting latest in_progress status from BigQuery:`, err.errors);
            } else if (err.code === 404) {
              sails.log.warn(`When an issue (${issueDetails.repo}#${issueDetails.issueNumber}) was moved into the "Ready for release" column, in progress data could not be found. BigQuery table or dataset not found. Please ensure the table exists:`, {
                dataset: 'github_metrics',
                table: 'issue_status_change',
                fullError: err.message
              });
            } else if (err.code === 403) {
              sails.log.warn(`When an issue (${issueDetails.repo}#${issueDetails.issueNumber}) was moved into the "Ready for release" column, in progress data could not be found. Permission denied when accessing BigQuery. Check service account permissions.`);
            } else {
              sails.log.warn(`When an issue (${issueDetails.repo}#${issueDetails.issueNumber}) was moved into the "Ready for release" column, There was an error getting latest in_progress status from BigQuery:`, err);
            }
            return null;
          });

          if (inProgressData) {
            // Calculate time to release ready (from in_progress to release)
            const releaseReadyTime = new Date(projectsV2Item.updated_at);  // Use webhook timestamp
            const inProgressTime = new Date(inProgressData.date);

            let timeToReleaseReadySeconds = await sails.helpers.flow.build(async ()=>{
              if (!sails.config.custom.githubProjectsV2.excludeWeekends) {
                // If weekend exclusion is disabled, return simple time difference
                return Math.floor((releaseReadyTime - inProgressTime) / 1000);
              }

              // Use the provided weekend exclusion logic
              const startDay = inProgressTime.getUTCDay();
              const endDay = releaseReadyTime.getUTCDay();

              // Case: Both start time and end time are on the same weekend
              if (
                (startDay === 0 || startDay === 6) &&
                (endDay === 0 || endDay === 6) &&
                Math.floor(releaseReadyTime / (24 * 60 * 60 * 1000)) -
                Math.floor(inProgressTime / (24 * 60 * 60 * 1000)) <=
                2
              ) {
                // Return 0 seconds
                return 0;
              }

              // Make copies to avoid modifying original dates
              const adjustedStartTime = new Date(inProgressTime);
              const adjustedEndTime = new Date(releaseReadyTime);

              // Set to start of Monday if start time is on weekend
              if (startDay === 0) {
                // Sunday
                adjustedStartTime.setUTCDate(adjustedStartTime.getUTCDate() + 1);
                adjustedStartTime.setUTCHours(0, 0, 0, 0);
              } else if (startDay === 6) {
                // Saturday
                adjustedStartTime.setUTCDate(adjustedStartTime.getUTCDate() + 2);
                adjustedStartTime.setUTCHours(0, 0, 0, 0);
              }

              // Set to start of Saturday if end time is on Sunday
              if (endDay === 0) {
                // Sunday
                adjustedEndTime.setUTCDate(adjustedEndTime.getUTCDate() - 1);
                adjustedEndTime.setUTCHours(0, 0, 0, 0);
              } else if (endDay === 6) {
                // Saturday
                adjustedEndTime.setUTCHours(0, 0, 0, 0);
              }

              // Count weekend days between adjusted dates
              // Make local copies for weekend counting
              let weekendStartDate = new Date(adjustedStartTime);
              let weekendEndDate = new Date(adjustedEndTime);

              // Ensure weekendStartDate is before weekendEndDate
              if (weekendStartDate > weekendEndDate) {
                [weekendStartDate, weekendEndDate] = [weekendEndDate, weekendStartDate];
              }

              // Make sure start dates and end dates are not on weekends. We just want to count the weekend days between them.
              if (weekendStartDate.getUTCDay() === 0) {
                weekendStartDate.setUTCDate(weekendStartDate.getUTCDate() + 1);
              } else if (weekendStartDate.getUTCDay() === 6) {
                weekendStartDate.setUTCDate(weekendStartDate.getUTCDate() + 2);
              }
              if (weekendEndDate.getUTCDay() === 0) {
                weekendEndDate.setUTCDate(weekendEndDate.getUTCDate() - 2);
              } else if (weekendEndDate.getUTCDay() === 6) {
                weekendEndDate.setUTCDate(weekendEndDate.getUTCDate() - 1);
              }

              let weekendDays = 0;
              const current = new Date(weekendStartDate);

              while (current <= weekendEndDate) {
                const day = current.getUTCDay();
                if (day === 0 || day === 6) {
                  // Sunday (0) or Saturday (6)
                  weekendDays++;
                }
                current.setUTCDate(current.getUTCDate() + 1);
              }

              // Calculate raw time difference in milliseconds
              const diffMs = adjustedEndTime - adjustedStartTime - weekendDays * 24 * 60 * 60 * 1000;

              // Ensure we don't return negative values
              return Math.max(0, Math.floor(diffMs / 1000));
            });

            // Determine project name by reverse lookup
            const projectName = Object.keys(sails.config.custom.githubProjectsV2.projects).find(
              key => sails.config.custom.githubProjectsV2.projects[key] === projectNumber
            ) || '';

            // Prepare release ready data
            const releaseReadyData = {
              release_ready: releaseReadyTime.toISOString().split('T')[0],  // eslint-disable-line camelcase
              assignee: issueDetails.assignee || '',  // Get assignee from issue details
              issue_url: `https://github.com/${issueDetails.repo}/issues/${issueDetails.issueNumber}`,  // eslint-disable-line camelcase
              time_to_release_ready_seconds: timeToReleaseReadySeconds,  // eslint-disable-line camelcase
              repo: issueDetails.repo,
              issue_number: issueDetails.issueNumber,  // eslint-disable-line camelcase
              release_ready_time: releaseReadyTime.toISOString(),  // eslint-disable-line camelcase
              in_progress_time: inProgressTime.toISOString(),  // eslint-disable-line camelcase
              project: projectName,
              type: issueDetails.type  // Issue type based on labels
            };

            // Save to BigQuery
            await sails.helpers.engineeringMetrics.saveToBigquery.with({
              data: releaseReadyData,
              gcpServiceAccountKey: gcpServiceAccountKey,
              tableId: 'issue_release_ready'
            });

            sails.log.verbose('Saved release ready metrics:', {
              repo: issueDetails.repo,
              issueNumber: issueDetails.issueNumber,
              timeToReleaseReadySeconds,
              project: projectName
            });

            statusChange = releaseReadyData;
          } else {
            sails.log.info(`No in_progress status found for ${issueDetails.repo}#${issueDetails.issueNumber}, cannot calculate release ready time`);
          }
        }
      }//ﬁ

      if (statusChange) {
        sails.log.verbose('Processed issue status change:', statusChange);
      }
    } else {
      //  ███╗   ███╗██╗███████╗ ██████╗
      //  ████╗ ████║██║██╔════╝██╔════╝
      //  ██╔████╔██║██║███████╗██║
      //  ██║╚██╔╝██║██║╚════██║██║
      //  ██║ ╚═╝ ██║██║███████║╚██████╗
      //  ╚═╝     ╚═╝╚═╝╚══════╝ ╚═════╝
      //
      // FUTURE: more potential stuff
      //
      // For reference:  (as of Apr 16, 2019)
      // Ping                  : (no "action" included)
      // Issue                 : opened, edited, deleted, transferred, pinned, unpinned, closed, reopened, assigned, unassigned, labeled, unlabeled, milestoned, demilestoned   (https://developer.github.com/v3/activity/events/types/#issuesevent)
      // PR                    : opened, closed, reopened, edited, assigned, unassigned, review requested, review request removed, labeled, unlabeled, synchronized, ready for review   (https://developer.github.com/v3/activity/events/types/#pullrequestevent)
      // Comment (pr or issue) : created, edited, deleted   (https://developer.github.com/v3/activity/events/types/#issuecommentevent)
      // Label                 : created, edited, deleted   (https://developer.github.com/v3/activity/events/types/#labelevent)
      // Commit comment        : created   (https://developer.github.com/v3/activity/events/types/#commitcommentevent)
      // PR review             : submitted, edited, dismissed   (https://developer.github.com/v3/activity/events/types/#pullrequestreviewevent
      // PR review comment     : created, edited, deleted   (https://developer.github.com/v3/activity/events/types/#pullrequestreviewcommentevent)
      // Branch, tag, or repo  : created, deleted   (https://developer.github.com/v3/activity/events/types/#createevent -- note thate "ref_type" can be either "tag", "branch", or "repository")
    }

  }


};
