module.exports = {


  friendlyName: 'Receive from zoom',


  description: 'Receive webhook requests and/or incoming auth redirects from Zoom.',


  inputs: {
    event: {
      type: 'string',
      required: true,
      isIn: [
        'endpoint.url_validation',
        'revenue_accelerator.conversation_analysis_completed',
      ],
    },
    event_ts: {// eslint-disable-line camelcase
      type: 'number',
    },
    payload: {
      type: {},
    }
  },


  exits: {
    success: { description: 'A webhook event has successfully been received.'},
    callInfoNotFound: {description: 'No information about this call could be found in the Zoom API.', responseType: 'badRequest'},
    callTranscriptNotFound: {description: 'No transcript for this call could be found in the Zoom API.', responseType: 'badRequest'},
    callHasNoTranscript: {description: 'The receive-from-zoom webhook received an event about a call with no transcript.', responseType: 'ok'},
    unexpectedEvent: {description: 'The receive-from-zoom webhook received an unsupported event.', responseType: 'badRequest'},
  },


  fn: async function ({ event, payload }) {

    require('assert')(sails.config.custom.zoomAccountId);
    require('assert')(sails.config.custom.zoomClientId);
    require('assert')(sails.config.custom.zoomClientSecret);
    require('assert')(sails.config.custom.zoomWebhookToken);

    if (!sails.config.custom.zoomWebhookSecret) {
      throw new Error('No Zoom webhook secret configured!  (Please set `sails.config.custom.zoomWebhookSecret`.)');
    }
    let webhookSecret = this.req.get('x-webhook-secret');
    if(webhookSecret !== sails.config.custom.zoomWebhookSecret) {
      return this.res.unauthorized();
    }


    if(event === 'endpoint.url_validation'){
      if(!payload.plainToken){
        sails.log.warn(`When the receive-from-zoom webhook recieved an event to validate the webhook URL, the provided payload did not contain a token. Full payload: ${require('util').inpsect(payload, {depth: null})}`);
        return this.res.badRequest();
      }
      // [?]: https://nodejs.org/docs/latest-v20.x/api/crypto.html#class-hmac
      let hmac = require('crypto').createHmac('sha256', sails.config.custom.zoomWebhookToken);
      hmac.update(payload.plainToken);
      let encryptedTokenToReturnToZoom = hmac.digest('hex');

      // Return the plainToken and encryptedToken to Zoom as JSON.
      return this.res.json({
        plainToken: payload.plainToken,
        encryptedToken: encryptedTokenToReturnToZoom
      });

    } else if(event === 'revenue_accelerator.conversation_analysis_completed'){


      // Get Zoom OAuth token:
      let oauthResponse = await sails.helpers.http.post.with({
        url: `https://zoom.us/oauth/token?grant_type=account_credentials&account_id=${sails.config.custom.zoomAccountId}`,
        headers: {
          'Authorization': `Basic ${Buffer.from(`${sails.config.custom.zoomClientId}:${sails.config.custom.zoomClientSecret}`).toString('base64')}`,
        },
        data: {
          grant_type: 'account_credentials',// eslint-disable-line camelcase
          account_id: sails.config.custom.zoomAccountId// eslint-disable-line camelcase
        }
      }).intercept((err)=>{
        return new Error(`When sending a request to get a Zoom access token, an error occured. Full error ${require('util').inspect(err, {depth: 3})}`);
      });
      let token = oauthResponse.access_token;

      let idOfCallToGenerateTranscriptFor = payload.object.conversation_id;
      let informationAboutThisCall = await sails.helpers.http.get.with({
        url: `https://api.zoom.us/v2/zra/conversations/${encodeURIComponent(idOfCallToGenerateTranscriptFor)}`,
        headers: {
          'Authorization': `Bearer ${token}`
        }
      })
      .intercept({raw: {statusCode: 404}}, (err)=>{
        sails.log.warn(`The receive-from-zoom webhook received an event (type: ${event}) about a Zoom call (id: ${idOfCallToGenerateTranscriptFor}), the Zoom API returned a 404 response when a request was sent to get information about the call. Full payload from Zoom: ${require('util').inpsect(payload, {depth: null})} Full error: ${require('util').inspect(err, {depth: 3})}`);
        return 'callInfoNotFound';
      }).intercept((err)=>{
        return new Error(`When sending a request to get information about a Zoom recording, an error occured. Full error ${require('util').inspect(err, {depth: 3})}`);
      });


      // Get a transcript of the call.
      let callTranscript = await sails.helpers.http.get.with({
        url: `https://api.zoom.us/v2/zra/conversations/${encodeURIComponent(idOfCallToGenerateTranscriptFor)}/interactions?page_size=300`,
        headers: {
          'Authorization': `Bearer ${token}`
        }
      })
      .intercept({raw: {statusCode: 404}}, (err)=>{
        sails.log.warn(`The receive-from-zoom webhook received an event (type: ${event}) about a Zoom call (id: ${idOfCallToGenerateTranscriptFor}), the Zoom API returned a 404 response when a request was sent to get a transcript of the call. Full payload from Zoom: ${require('util').inpsect(payload, {depth: null})} Full error: ${require('util').inspect(err, {depth: 3})}`);
        return 'callTranscriptNotFound';
      }).intercept((err)=>{
        return new Error(`When sending a request to get a transcript of a Zoom recording, an error occured. Full error ${require('util').inspect(err, {depth: 3})}`);
      });

      // If a call is recorded, but a user stops the recording before anything is said, the call interactions will not have any participants.
      // If this happens, we'll throw callTranscriptNotFound exit, which returns a 200 response to Zoom.
      if(!callTranscript.participants) {
        throw 'callHasNoTranscript';
      }

      let allSpeakersOnThisCall = [];
      allSpeakersOnThisCall = allSpeakersOnThisCall.concat(callTranscript.participants);
      let tokenForNextPageOfResults = callTranscript.next_page_token;
      // If a next_page_token was provided in the response body, we do not have all of the transcript.
      if(tokenForNextPageOfResults) {
        await sails.helpers.flow.until(async()=>{
          let thisPageOfCallInformation = await sails.helpers.http.get.with({
            url: `https://api.zoom.us/v2/zra/conversations/${encodeURIComponent(idOfCallToGenerateTranscriptFor)}/interactions?next_page_token=${encodeURIComponent(tokenForNextPageOfResults)}`,
            headers: {
              'Authorization': `Bearer ${token}`
            }
          }).intercept((err)=>{
            return new Error(`When the receive-from-zoom webhook send a request to get a page of a call transcript (call id: ${idOfCallToGenerateTranscriptFor}) an error occured. Full error: ${require('util').inspect(err, {depth: null})}`);
          });
          allSpeakersOnThisCall = allSpeakersOnThisCall.concat(thisPageOfCallInformation.participants);
          tokenForNextPageOfResults = thisPageOfCallInformation.next_page_token;
          // Stop the until() helper when the response body does not contain a token for the next page of results.
          return thisPageOfCallInformation.next_page_token === '';
        }).intercept((err)=>{
          return new Error(`When the receive-from-zoom webhook attempted to process multiple pages of a call transcript (call ID: ${idOfCallToGenerateTranscriptFor}). An error occured. full error ${require('util').inspect(err, {depth: null})}`);
        });
      }

      // Transcripts are ordered by an item_id, but separaterd by speaker.
      let allTranscriptLines = [];
      for(let speaker of allSpeakersOnThisCall) {
        if(!speaker.transcripts) {
          // Log a warning if a speaker on this call is missing a transcripts value, and proceed.
          sails.log.warn(`When the receive-from-zoom webhook attempted to create a call transcript of a Zoom meeting (call id: ${idOfCallToGenerateTranscriptFor}), a speaker on the call is missing an expected 'transcripts' value. Speaker information returned from the Zoom API: ${require('util').inspect(speaker)}`);
          continue;
        }
        for(let line of speaker.transcripts) {
          // Rebuild a list of lines in the call transcript and attach the speakers name to eac hline in the transcript
          allTranscriptLines.push({
            id: Number(line.item_id),
            text: line.text,
            speaker: speaker.display_name,
          });
        }
      }
      let allSpokenWordsOrderedById = _.sortBy(allTranscriptLines, 'id');

      let transcript = '';
      let lastSpeaker;
      // Now iterate through the ordered list of transcript lines and build a full transcript.
      for(let line of allSpokenWordsOrderedById) {
        if(line.speaker !== lastSpeaker){
          transcript += `\n${line.speaker}:\n${line.text}\n`;
        } else {
          transcript += `${line.text}\n`;
        }
        lastSpeaker = line.speaker;
      }

      // Send a POST request to Zapier with the transcript and information about this recording.
      await sails.helpers.http.post.with({
        url: 'https://hooks.zapier.com/hooks/catch/3627242/2lp3acb/',
        data: {
          transcript: transcript,
          topic: informationAboutThisCall.topic,
          participants: _.pluck(allSpeakersOnThisCall, 'display_name').join(', '),
          participantEmails: _.pluck(allSpeakersOnThisCall, 'email').join(', '),
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
    } else {
      // Otherwise, return an unexpectedEvent response.
      throw 'unexpectedEvent';
    }
    return;

  }


};
