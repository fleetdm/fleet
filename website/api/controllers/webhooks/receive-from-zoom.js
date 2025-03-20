module.exports = {


  friendlyName: 'Receive from zoom',


  description: 'Receive webhook requests and/or incoming auth redirects from Zoom.',


  inputs: {
    webhookSecret: {
      type: 'string',
      required: true
    },
    event: {
      type: 'string',
      isIn: [
        'revenue_accelerator.conversation_analysis_completed',
      ],
    },
    payload: {
      type: {
        account_id: 'string',
        object: {
          conversation_id: 'string',
          source: 'string',
        }
      }
    }
  },


  exits: {

  },


  fn: async function ({webhookSecret, event, payload }) {

    if(webhookSecret !== sails.config.custom.zoomWebhookSecret) {
      return this.res.unauthorized();
    }

    // Get zoom OAuth token:
    let oauthResponse = await sails.helpers.http.post.with({
      url: `https://zoom.us/oauth/token?grant_type=account_credentials&account_id=${sails.config.custom.zoomAccountId}`,
      headers: {
        'Authorization': `Basic ${Buffer.from(`${sails.config.custom.zoomClientId}:${sails.config.custom.zoomClientSecret}`).toString('base64')}`,
      },
      data: {
        grant_type: 'account_credentials',
        account_id: sails.config.custom.zoomAccountId
      }
    });
    let token = oauthResponse.access_token;


    let idOfCallToGenerateTranscriptFor = payload.object.conversation_id;
    let informationAboutThisCall = await sails.helpers.http.get.with({
      url: `https://api.zoom.us/v2/zra/conversations/${encodeURIComponent(idOfCallToGenerateTranscriptFor)}`,
      headers: {
        'Authorization': `Bearer ${token}`
      }
    });
    // console.log('call info \n ________')
    // console.log(informationAboutThisCall);
    // console.log('________')
    // Get a transcript of the call.
    let callTranscript = await sails.helpers.http.get.with({
      url: `https://api.zoom.us/v2/zra/conversations/${encodeURIComponent(idOfCallToGenerateTranscriptFor)}/interactions?page_size=300`,
      headers: {
        'Authorization': `Bearer ${token}`
      }
    });
    // // TODO: If a next_page_token was provided, we need to get more pages.
    // if(callTranscript.next_page_token) {

    // }
    // console.log('call transcript \n ________')
    // console.log(callTranscript);
    // console.log('________')
    // // Get a transcript of a call
    // // console.log(callInfo);
    // console.log(`building complete transcript for ${informationAboutThisCall.topic}`);

    // Transcripts are ordered by an item_id, but separaterd by speaker.
    let allTranscriptLines = [];
    for(let speaker of callTranscript.participants) {
      for(let line of speaker.transcripts){
        // Rebuild a list of lines in the call transcript and attach the speakers name to eac hline in the transcript
        allTranscriptLines.push({
          id: line.item_id,
          text: line.text,
          speaker: speaker.display_name,
        });
      }
    }

    let allSpokenWordsOrderedById = _.sortBy(allTranscriptLines, 'id');

    let transcript = '';
    // Now iterate through the ordered list of transcript lines and build a full transcript.
    for(let line of allSpokenWordsOrderedById){
      transcript += `${line.speaker}: \n ${line.text} \n\n`;
    }


    // Send a POST request to Zapier with the release object.
    await sails.helpers.http.post.with({
      url: 'https://hooks.zapier.com/hooks/catch/3627242/2lp3acb/',
      data: {
        transcript: transcript,
        topic: informationAboutThisCall.topic,
        participants: _.pluck(callTranscript.participants, 'display_name').join(','),
        zoomUrl: informationAboutThisCall.conversation_url,
        startTime: informationAboutThisCall.meeting_start_time,
        webhookSecret: sails.config.custom.zapierSandboxWebhookSecret,
      }
    })
    .timeout(5000)
    .tolerate(['non200Response', 'requestFailed', {name: 'TimeoutError'}], (err)=>{
      // Note that Zapier responds with a 2xx status code even if something goes wrong, so just because this message is not logged doesn't mean everything is hunky dory.  More info: https://github.com/fleetdm/fleet/pull/6380#issuecomment-1204395762
      sails.log.warn(`When trying to send a Zoom transcript to Zapier, an error occured. Raw error: ${require('util').inspect(err)}`);
      return;
    });
    return;

  }


};
