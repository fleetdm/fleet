module.exports = {


  friendlyName: 'View workshops',


  description: 'Display "Workshops" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/workshops'
    }

  },


  fn: async function () {

    // Retrieve the single platform record from the database.
    let platformRecords = await Platform.find();
    let platformRecord = platformRecords[0];
    if(!platformRecord) {
      throw new Error(`Consistency violation: when a user visited the workshops page, no platform record was found.`);
    }

    // Check the workshopDetailsLastUpdatedAt timestamp on the platform record
    let nowAt = Date.now();
    let twoHoursAgoAt = nowAt - (1000 * 60 * 60 * 2);
    let eventDetailsWereUpdatedLessThanTwoHoursAgo = (twoHoursAgoAt > platformRecord.workshopDetailsLastUpdatedAt);

    // Get the event details from the platform record.
    let eventDetails = platformRecord.workshopDetails;

    let futureWorkshops = [];

    // If the platform record was updated in the past two hours, use the event details stored in the website's database.
    if(eventDetails.length > 1 && !eventDetailsWereUpdatedLessThanTwoHoursAgo) {
      futureWorkshops = eventDetails;
    } else {// Otherwise, fetch fresh data from the EventBrite API
      let futureGitopsEvents = await sails.helpers.http.get.with({
        url: `https://www.eventbriteapi.com/v3/organizations/${sails.config.custom.eventbriteOrgId}/events/?show_series_parent=true&page_size=200&time_filter=current_future&status=live`,
        headers: {
          authorization: `Bearer ${sails.config.custom.eventbriteApiToken}`
        },
      }).tolerate((err)=>{
        sails.log.warn(`When a user visited the gitops workshop page, a list of future gitiops workshops could not be obtained from the Eventbrite API. Full error: ${require('util').inspect(err)}`);
        return {
          events: [],
        };
      });

      let eventsToGetDetailsFor = futureGitopsEvents.events;

      await sails.helpers.flow.simultaneouslyForEach(eventsToGetDetailsFor, async (event)=>{
        // Determine if this event is a GitOps workshop or an Apple administrator workshop.
        let eventType = _.contains(event.name.text.toLowerCase(), 'gitops') ? 'GitOps workshop' : _.contains(event.name.text.toLowerCase(), 'apple administrator') ? 'Apple administrator workshop' : undefined;
        // If it is not one of those types of events, skip it.
        if(!eventType) {
          return;
        }

        // Convert the ISO timestamps that represent the start and end time of the event into a formatted string.
        // Create new Date objects from the start and end times.
        let eventStartsOn = new Date(event.start.utc);
        let eventEndsOn = new Date(event.end.utc);
        // Get a JS timestamp of when this event starts (used to sort the final list of events.)
        let eventStartsAt = eventStartsOn.getTime();
        let eventTimeZone = event.start.timezone;
        let formattedDateString = new Intl.DateTimeFormat('en-US', {
          timeZone: eventTimeZone,
          month: 'short',
          day: 'numeric',
        }).format(eventStartsOn);
        // ex: 2026-02-03T13:00:00 »»» Feb 3

        // Create a new Intl.DateTimeFormat object using the start Date object, and use the formatToParts() method to get an abbreviated timezone string.
        let partsOfTimezone = new Intl.DateTimeFormat('en-US', {
          timeZone: event.start.timezone,
          timeZoneName: 'short'
        }).formatToParts(eventStartsOn);// [?]: https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Intl/DateTimeFormat/formatToParts

        // Get the value of the timeZoneName object in the partsOfTimezone array/
        let shortenedTimeZone = _.find(partsOfTimezone, {type: 'timeZoneName'});
        let abbreviatedTimeZoneString = shortenedTimeZone.value;

        let startTime =  new Intl.DateTimeFormat('en-US', {
          timeZone: eventTimeZone,
          hour: 'numeric',
          minute: 'numeric',
          hour12: true
        })
        .format(eventStartsOn)
        .replace(':00 ', '')
        .toLowerCase();

        let endTime =  new Intl.DateTimeFormat('en-US', {
          timeZone: eventTimeZone,
          hour: 'numeric',
          minute: 'numeric',
          hour12: true
        })
        .format(eventEndsOn)
        .replace(':00 ', '')
        .toLowerCase();
        let eventTimeDetailsString = `${formattedDateString} from ${startTime} to ${endTime} ${abbreviatedTimeZoneString}`;
        let eventDetails = {
          eventbriteLink: event.url,
          eventTime: eventTimeDetailsString,
          startsAt: eventStartsAt,
          type: eventType,
        };

        // If an event has a venue_id, we'll send a request to the Eventbrite API to get details about the venue.
        if(event.venue_id) {
          let eventVenueResponse = await sails.helpers.http.get.with({
            url: `https://www.eventbriteapi.com/v3/venues/${event.venue_id}/`,
            headers: {
              authorization: `Bearer ${sails.config.custom.eventbriteApiToken}`
            },
          }).tolerate((err)=>{
            sails.log.warn(`When a user visited the gitops workshop page, details about a venue for an event (${event.name.text}) could not be obtained from the Eventbrite API. Full error: ${require('util').inspect(err)}`);
            // If there was an error getting details about the venue for this event, set the address to 'TBA' and use the event name instead of the city name.
            return {
              address: { city: event.name.text },
              name: 'TBA'
            };
          });
          eventDetails.workshopCity = eventVenueResponse.address.city;
          eventDetails.workshopAddress = eventVenueResponse.name;
        } else {
          // IF the event is missing a venue_id, set the address to 'TBA' and use the event name instead of the city name.
          eventDetails.workshopCity = event.name.text;
          eventDetails.workshopAddress = 'TBA';
        }


        futureWorkshops.push(eventDetails);
      });


      // Update the plaform record with the latest workshop event details
      await Platform.updateOne({id: platformRecord.id}).set({workshopDetails: futureWorkshops, workshopDetailsLastUpdatedAt: Date.now()});
    }



    // Sort the events that will be displayed on the page.
    futureWorkshops = _.sortBy(futureWorkshops, 'startsAt');
    // Respond with view.
    return {
      futureWorkshops
    };

  }


};
