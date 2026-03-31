module.exports = {


  friendlyName: 'View gitops workshop',


  description: 'Display "Gitops workshop" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/landing-pages/gitops-workshop'
    }

  },


  fn: async function () {

    let futureGitopsWorkshops = [];


    let futureGitopsEvents = await sails.helpers.http.get.with({
      url: `https://www.eventbriteapi.com/v3/organizations/${sails.config.custom.eventbriteOrgId}/events/?name_filter=gitops&show_series_parent=true&page_size=200&time_filter=current_future&status=live`,
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


      futureGitopsWorkshops.push(eventDetails);
    });
    // Sort the events that will be displayed on the page.
    futureGitopsWorkshops = _.sortBy(futureGitopsWorkshops, 'startsAt');
    // Respond with view.
    return {
      futureGitopsWorkshops
    };

  }


};
