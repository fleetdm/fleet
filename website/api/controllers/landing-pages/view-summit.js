module.exports = {


  friendlyName: 'View summit',


  description: 'Display "Summit" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/landing-pages/summit'
    }

  },


  fn: async function () {

    // The "Register on LinkedIn" buttons are hidden until this is set to the LinkedIn event URL.
    let registrationUrl = '';

    let startsAt = '2026-11-10T16:00:00Z';
    let endsAt = '2026-11-10T18:00:00Z';

    // Speakers without a headshot show their initials. To add one, set `imageSrc` to a 48x48@2x image in /images.
    let sessions = [
      {
        topic: 'Infrastructure as code',
        title: 'Device management finally gets a deployment pipeline',
        description: 'Engineering teams version-control production infrastructure, review changes in pull requests, and roll back with a merge. Device management is one of the last IT functions still changed by hand in a console. Practitioners who moved their device configuration into Git share what it unlocked for shipping changes, compliance evidence, and recovering from mistakes.',
        speakers: [
          { name: 'Adam Anklewicz', company: 'Thumbtack' },
          { name: 'Brock Walters', company: 'Treeline' },
          { name: 'Viktor Filipsson', company: 'Sonos' },
          { name: 'Betsy Keiser', company: 'SandboxAQ' },
        ],
        moreSpeakersToBeAnnounced: false,
      },
      {
        topic: 'Open source',
        title: 'Why open source ate the enterprise fleet',
        description: 'Open source runs most enterprise infrastructure, but the tools that manage employee devices are still mostly closed. This panel covers why organizations choose auditable code and portable data, and how that choice plays out in procurement, security reviews, and vendor negotiations.',
        speakers: [
          { name: 'Daniel Moore', company: 'Red Hat' },
          { name: 'Jarryd Stanbrook', company: 'Easygo' },
          { name: 'Patricia Egger', company: 'Proton' },
        ],
        moreSpeakersToBeAnnounced: false,
      },
      {
        topic: 'AI agents',
        title: 'AI agents need endpoints too',
        description: 'AI agents are showing up on employee devices faster than IT can inventory them. They read files, call APIs, and act with the permissions of whoever installed them. At the same time, AI-driven exploits are shrinking the time between disclosure and attack. This panel covers both sides of the problem.',
        speakers: [
          { name: 'Dustin Davis', company: 'Pinterest' },
          { name: 'Jason Walton', company: 'Schrödinger' },
        ],
        moreSpeakersToBeAnnounced: true,
      },
      {
        topic: 'Device management 2030',
        title: 'Device management 2030',
        description: 'The devices IT manages, the tools it uses, and the skills the job requires will look different four years from now. Practitioners who are already placing bets on that future talk about where IT teams, tools, and the definition of an endpoint are headed.',
        speakers: [
          { name: 'Mohammed Saqr', company: 'Block' },
          { name: 'Dave Hannigan', company: 'Former CISO, Nubank' },
        ],
        moreSpeakersToBeAnnounced: true,
      },
    ];

    let panelistCompanies = ['Thumbtack', 'Treeline', 'Sonos', 'SandboxAQ', 'Red Hat', 'Easygo', 'Proton', 'Pinterest', 'Schrödinger', 'Block'];

    let googleCalendarUrl = 'https://calendar.google.com/calendar/render?' + new URLSearchParams({
      action: 'TEMPLATE',
      text: 'Fleet Virtual Summit 2026',
      dates: startsAt.replace(/[-:]/g, '') + '/' + endsAt.replace(/[-:]/g, ''),
      details: 'Four panel conversations on how GitOps, open source, and AI agents get device management to 2030. Streaming free on LinkedIn Live. Details: '+sails.config.custom.baseUrl+'/summit',
      location: registrationUrl || sails.config.custom.baseUrl+'/summit',
    }).toString();

    // Respond with view.
    return {
      registrationUrl,
      startsAt,
      endsAt,
      sessions,
      panelistCompanies,
      googleCalendarUrl,
    };

  }


};
