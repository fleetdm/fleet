parasails.registerPage('summit-page', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    localStartTime: '',
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    let timeZone = Intl.DateTimeFormat().resolvedOptions().timeZone;
    // Only show the viewer's local time when it isn't already one of the listed time zones.
    if (!['America/New_York', 'America/Los_Angeles', 'Europe/London'].includes(timeZone)) {
      this.localStartTime = new Date(this.startsAt).toLocaleString(undefined, { weekday: 'short', month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit', timeZoneName: 'short' });
    }
  },
  mounted: async function() {
    //…
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    clickDownloadCalendarFile: function() {
      let toIcsTimestamp = (isoString)=>{ return isoString.replace(/[-:]/g, '').replace(/\.\d+/, ''); };
      let pageUrl = window.location.origin + '/summit';
      let icsContents = [
        'BEGIN:VCALENDAR',
        'VERSION:2.0',
        'PRODID:-//Fleet//Fleet Virtual Summit//EN',
        'BEGIN:VEVENT',
        'UID:fleet-virtual-summit-2026@fleetdm.com',
        'DTSTAMP:' + toIcsTimestamp(new Date().toISOString()),
        'DTSTART:' + toIcsTimestamp(this.startsAt),
        'DTEND:' + toIcsTimestamp(this.endsAt),
        'SUMMARY:Fleet Virtual Summit 2026',
        'DESCRIPTION:Four panel conversations on how GitOps\\, open source\\, and AI agents get device management to 2030. Streaming free on LinkedIn Live. Details: ' + pageUrl,
        'LOCATION:' + (this.registrationUrl || pageUrl),
        'URL:' + (this.registrationUrl || pageUrl),
        'END:VEVENT',
        'END:VCALENDAR',
      ].join('\r\n');
      let downloadLink = document.createElement('a');
      downloadLink.href = URL.createObjectURL(new Blob([icsContents], { type: 'text/calendar' }));
      downloadLink.download = 'fleet-virtual-summit-2026.ics';
      document.body.appendChild(downloadLink);
      downloadLink.click();
      downloadLink.remove();
      URL.revokeObjectURL(downloadLink.href);
    },
  }
});
