module.exports = {


  friendlyName: 'View press',


  description: 'Display "Press" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/press'
    }

  },


  fn: async function () {

    // Press coverage entries. Add new items at the top of this list.
    // Each item must include: date (YYYY-MM-DD), publication, title, url.
    // Optional: featured (boolean) marks an entry as the lead/featured story.
    let pressCoverageEntries = [
      { date: '2026-05-26', publication: 'e-Channel News',         title: 'Inside Fleet\'s 100% transition to a partner-first go-to-market',                                                                     url: 'https://www.e-channelnews.com/inside-fleets-100-transition-to-a-partner-first-go-to-market/' },
      { date: '2026-05-15', publication: 'Solutions Review',       title: 'Endpoint security and network monitoring news for the week of May 15th',                                                              url: 'https://solutionsreview.com/endpoint-security/endpoint-security-and-network-monitoring-news-for-the-week-of-may-15th/' },
      { date: '2026-05-15', publication: 'Cybersecurity Insiders', title: 'Fleet delivers Mythos-ready endpoint management automation',                                                                          url: 'https://www.cybersecurity-insiders.com/fleet-delivers-mythos-ready-endpoint-management-automation/' },
      { date: '2026-05-15', publication: 'Cyber Defense Wire',     title: 'Fleet unveils Mythos-ready autonomous endpoint management',                                                                           url: 'https://cyberdefensewire.com/fleet-unveils-mythos-ready-autonomous-endpoint-management/' },
      { date: '2026-05-14', publication: 'VMBlog',                 title: 'Fleet delivers Mythos-ready endpoint management automation',                                                                          url: 'https://vmblog.com/news/fleet-delivers-mythos-ready-endpoint-management-automation/' },
      { date: '2026-05-14', publication: 'SC Media',               title: 'Fleet Device Management launches autonomous endpoint management platform',                                                            url: 'https://www.scworld.com/brief/fleet-device-management-launches-autonomous-endpoint-platform-amid-ai-exploit-concerns' },
      { date: '2026-05-14', publication: 'SiliconANGLE',           title: 'Fleet launches autonomous endpoint management platform to counter AI-accelerated exploits',                                           url: 'https://siliconangle.com/2026/05/14/fleet-launches-autonomous-endpoint-management-platform-counter-ai-accelerated-exploits/' },
      { date: '2026-05-14', publication: 'Channel Insider',        title: 'Fleet CEO: faster remediations need IT and partner support',                                                                          url: 'https://www.channelinsider.com/security/tools-and-platforms/fleet-autonomous-endpoint-management-patching/' },
      { date: '2026-05-14', publication: 'TechStrong',             title: 'Fleet adds ability to automate deployment of software patches and updates',                                                           url: 'https://techstrong.it/featured/fleet-adds-ability-to-automate-deployment-of-software-patches-and-updates/' },
      { date: '2026-05-14', publication: 'PR Newswire',            title: 'Fleet launches Mythos-ready autonomous endpoint management',                                                                          url: 'https://www.prnewswire.com/news-releases/fleet-launches-mythos-ready-autonomous-endpoint-management-302771551.html' },
      { date: '2026-05-14', publication: 'Yahoo Finance',          title: 'Fleet launches Mythos-ready autonomous endpoint management',                                                                          url: 'https://finance.yahoo.com/sectors/technology/articles/fleet-launches-mythos-ready-autonomous-130000836.html' },
      { date: '2026-05-14', publication: 'StockTitan',             title: 'As AI speeds up hacks, Fleet turns employee devices into self-patchers',                                                              url: 'https://www.stocktitan.net/news/FSLY/fleet-launches-mythos-ready-autonomous-endpoint-o4wvvxntq3uy.html' },
      { date: '2026-04-29', publication: 'Computerworld',          title: 'Fleet hopes to be the MDM provider for the AI era',                                                                                   url: 'https://www.computerworld.com/article/4164402/fleet-hopes-to-be-the-mdm-provider-for-the-ai-era.html' },
      { date: '2026-04-28', publication: 'Channel Insider',        title: 'Fleet targets MSPs and resellers with new partner program',                                                                           url: 'https://www.channelinsider.com/channel-business/vendor-leadership-and-partner-programs/fleet-partner-program-device-management/' },
      { date: '2026-04-23', publication: 'Channel Dive',           title: 'Fleet goes partner-first as device management gets complicated',                                                                      url: 'https://www.channeldive.com/news/fleet-goes-partner-first-as-device-management-gets-complicated/818218/' },
      { date: '2026-04-20', publication: 'ChannelBuzz',            title: 'The Buzz: Fleet goes 100 percent channel',                                                                                             url: 'https://channelbuzz.ca/2026/04/the-buzz-fleet-goes-100-percent-channel-scale-computing-revamps-partner-program-and-n-ables-ai-advice-46335/' },
      { date: '2026-04-20', publication: 'CRN',                    title: 'Five companies that came to win this week',                                                                                            url: 'https://www.crn.com/news/channel-news/2026/five-companies-that-came-to-win-this-week-april-17-2026' },
      { date: '2026-04-17', publication: 'Cyber Defense Wire',     title: 'Fleet announces new partner program and names MobileIron co-founder Suresh Batchu to board',                                          url: 'https://cyberdefensewire.com/fleet-announces-new-partner-program-and-names-mobileiron-co-founder-suresh-batchu-to-board/' },
      { date: '2026-04-17', publication: 'Channele2e',             title: 'Channel Brief: MSP growth is getting harder to win',                                                                                   url: 'https://www.channele2e.com/news/channel-brief-its-less-about-tools-more-about-running-them' },
      { date: '2026-04-16', publication: 'CRN',                    title: 'Fleet launches inaugural partner program as it adopts a 100 percent channel sales model: Exclusive',                                  url: 'https://www.crn.com/news/channel-news/2026/fleet-launches-inaugural-partner-program-as-it-adopts-a-100-percent-channel-sales-model-exclusive' },
      { date: '2026-04-16', publication: 'Channelvision',          title: 'Fleet launches partner program for its device management',                                                                            url: 'https://channelvisionmag.com/fleet-launches-partner-program-for-its-device-management/' },
      { date: '2026-04-16', publication: 'Apple Must',             title: 'Fleet launches partner program, appoints MobileIron co-founder to its board',                                                         url: 'https://www.applemust.com/fleet-launches-partner-program-appoints-mobileiron-co-founder-to-its-board/' },
      { date: '2026-04-16', publication: 'PR Newswire',            title: 'Fleet launches partner program, appoints device management category pioneer to board',                                                url: 'https://www.prnewswire.com/news-releases/fleet-launches-partner-program-appoints-device-management-category-pioneer-to-board-302743614.html' },
      { date: '2026-04-16', publication: 'StockTitan',             title: 'Fleet launches partner program, adds Batchu to board',                                                                                url: 'https://www.stocktitan.net/news/MOBL/fleet-launches-partner-program-appoints-device-management-category-s4y9d8dwmo34.html' },
      { date: '2025-06-19', publication: 'FinSMEs',                title: 'Fleet raises $27M in Series B funding',                                                                                                url: 'https://www.finsmes.com/2025/06/fleet-raises-27m-in-series-b-funding.html' },
      { date: '2025-06-17', publication: '9to5Mac',                title: 'Fleet lands $27M Series B to expand open device management with cloud and self-hosting flexibility',                                  url: 'https://9to5mac.com/2025/06/17/fleet-lands-27m-series-b-to-expand-open-device-management-with-cloud-and-self-hosting-flexibility/' },
      { date: '2025-06-17', publication: 'SiliconANGLE',           title: 'With $27M in funding, Fleet wants to bring more freedom to enterprise device management',                                             url: 'https://siliconangle.com/2025/06/17/27m-funding-fleet-wants-bring-freedom-enterprise-device-management/' },
      { date: '2025-06-17', publication: 'Business Wire',          title: 'Fleet adds $27M to usher in new era of open device management',                                                                       url: 'https://www.businesswire.com/news/home/20250617550974/en/Fleet-Adds-27M-to-Usher-in-New-Era-of-Open-Device-Management' },
      { date: '2025-06-17', publication: 'Yahoo Finance',          title: 'Fleet adds $27M to usher in new era of open device management',                                                                       url: 'https://finance.yahoo.com/news/fleet-adds-27m-usher-era-140000489.html' },
      { date: '2025-06-17', publication: 'CTOL Digital',           title: 'Fleet raises $27 million in Series B funding to expand open-source device management platform',                                       url: 'https://www.ctol.digital/news/fleet-raises-27-million-series-b-funding-open-source-device-management/' },
      { date: '2025-06-17', publication: 'Ten Eleven Ventures',    title: 'Fleet adds $27M to usher in new era of open device management',                                                                       url: 'https://www.1011vc.com/news/fleet-adds-27m-to-usher-in-new-era-of-open-device-management/' },
      { date: '2025-06-17', publication: 'VCNewsDaily',            title: 'Fleet closes $27M Series B round',                                                                                                    url: 'https://vcnewsdaily.com/fleet-dm/venture-capital-funding/jbxmhmxdzm' },
      { date: '2022-04-28', publication: 'TechCrunch',             title: 'Fleet nabs $20M to enable enterprises to manage their devices',                                                                       url: 'https://techcrunch.com/2022/04/28/fleet-nabs-20m-to-enable-enterprises-to-manage-their-devices' },
      { date: '2022-04-28', publication: 'GlobeNewswire',          title: 'Fleet reaches 1.65 million devices enrolled, raises Series A at a $100M valuation for open source device management',                 url: 'https://www.globenewswire.com/en/news-release/2022/04/28/2431771/0/en/Fleet-Reaches-1-65-Million-Devices-Enrolled-Raises-Series-A-at-a-100M-Valuation-for-Open-Source-Device-Management.html' },
      { date: '2022-01-20', publication: 'VentureBeat',            title: 'How Fleet brings open source to enterprise device management',                                                                        url: 'https://venturebeat.com/business/how-fleet-brings-open-source-to-enterprise-device-management' },
    ];

    // Sort entries newest first.
    pressCoverageEntries.sort((a, b)=>{
      if (a.date < b.date) { return 1; }
      if (a.date > b.date) { return -1; }
      return 0;
    });

    // Group entries by year then by month (newest first within each).
    let monthNames = ['January', 'February', 'March', 'April', 'May', 'June', 'July', 'August', 'September', 'October', 'November', 'December'];
    let groupedByYear = [];
    for (let entry of pressCoverageEntries) {
      let parts = entry.date.split('-');
      let year = parseInt(parts[0], 10);
      let monthIndex = parseInt(parts[1], 10) - 1;
      let day = parseInt(parts[2], 10);
      let formattedDate = monthNames[monthIndex] + ' ' + day + ', ' + year;
      let shortDate = monthNames[monthIndex].substring(0, 3).toUpperCase() + ' ' + day;

      let yearGroup = _.find(groupedByYear, { year: year });
      if (!yearGroup) {
        yearGroup = { year: year, months: [], count: 0 };
        groupedByYear.push(yearGroup);
      }
      yearGroup.count += 1;

      let monthGroup = _.find(yearGroup.months, { name: monthNames[monthIndex] });
      if (!monthGroup) {
        monthGroup = { name: monthNames[monthIndex], monthIndex: monthIndex, items: [] };
        yearGroup.months.push(monthGroup);
      }

      monthGroup.items.push({
        date: entry.date,
        formattedDate: formattedDate,
        shortDate: shortDate,
        publication: entry.publication,
        title: entry.title,
        url: entry.url,
      });
    }

    // Sort years and months newest first.
    groupedByYear.sort((a, b)=> b.year - a.year);
    for (let yearGroup of groupedByYear) {
      yearGroup.months.sort((a, b)=> b.monthIndex - a.monthIndex);
    }

    // What's new: one item per month, newest first, capped at 4.
    let whatsNewItems = [];
    for (let yearGroup of groupedByYear) {
      for (let monthGroup of yearGroup.months) {
        if (whatsNewItems.length >= 4) { break; }
        if (monthGroup.items.length === 0) { continue; }
        let top = monthGroup.items[0];
        whatsNewItems.push({
          publication: top.publication,
          title: top.title,
          url: top.url,
          formattedDate: top.formattedDate,
          shortDate: top.shortDate,
          monthLabel: monthGroup.name + ' ' + yearGroup.year,
        });
      }
      if (whatsNewItems.length >= 4) { break; }
    }

    // Unique publications (used in stats and "as featured in" strip).
    let allPublications = _.uniq(_.pluck(pressCoverageEntries, 'publication'));
    let publicationsCount = allPublications.length;

    // Curated list shown in the "As featured in" strip (highest-recognition outlets first).
    let featuredPublicationOrder = ['TechCrunch', 'VentureBeat', 'Computerworld', 'CRN', '9to5Mac', 'SC Media', 'SiliconANGLE', 'Channel Insider', 'Cybersecurity Insiders', 'TechStrong', 'Solutions Review'];
    let featuredPublications = _.intersection(featuredPublicationOrder, allPublications);

    // Coverage span string for the hero stat row.
    let coverageSpan = '';
    if (pressCoverageEntries.length > 0) {
      let newest = pressCoverageEntries[0].date.split('-');
      let oldest = pressCoverageEntries[pressCoverageEntries.length - 1].date.split('-');
      let newestLabel = monthNames[parseInt(newest[1], 10) - 1] + ' ' + newest[0];
      let oldestLabel = monthNames[parseInt(oldest[1], 10) - 1] + ' ' + oldest[0];
      coverageSpan = (newestLabel === oldestLabel) ? newestLabel : oldestLabel + ' to ' + newestLabel;
    }

    return {
      pressCoverageByYear: groupedByYear,
      whatsNewItems: whatsNewItems,
      featuredPublications: featuredPublications,
      totalStories: pressCoverageEntries.length,
      publicationsCount: publicationsCount,
      coverageSpan: coverageSpan,
      mediaContactName: 'Alyssa Pallotti',
      mediaContactRole: 'Media relations',
      mediaContactEmail: 'alyssa@apt-pr.com',
    };

  }


};
