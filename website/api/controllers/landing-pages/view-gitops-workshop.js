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
      let eventVenueResponse = await sails.helpers.http.get.with({
        url: `https://www.eventbriteapi.com/v3/venues/${event.venue_id}/`,
        headers: {
          authorization: `Bearer ${sails.config.custom.eventbriteApiToken}`
        },
      }).tolerate((err)=>{
        sails.log.warn(`When a user visited the gitops workshop page, details about a venue for an event (${event.name.text}) could not be obtained from the Eventbrite API. Full error: ${require('util').inspect(err)}`);
        return {
          events: [],
        };
      });

      // Convert the ISO timestamps that represent the start and end time of the event into a formatted string.
      // Create new Date objects from the start and end times.
      let eventStartsOn = new Date(event.start.local);
      let eventEndsOn = new Date(event.end.local);

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
        workshopCity: eventVenueResponse.address.city,
        workshopAddress: eventVenueResponse.name,
        eventbriteLink: event.url,
        eventTime: eventTimeDetailsString,
      };
      futureGitopsWorkshops.push(eventDetails);
    });

    // Respond with view.
    return {
      futureGitopsWorkshops
    };

  }


};
