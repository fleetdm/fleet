module.exports = {


  friendlyName: 'Update cve',


  description: '',


  fn: async function () {



    let parseCsv = require('csv-parse/lib/sync');
    let rawCsvString = await sails.helpers.fs.read('cve-publish-dates.csv')
    .intercept((err) => new Error('The CSV file of CVE publish dates must be located in this repo (for now- until an equivalent URL is used instead).  Details: '+err.message));

    let parsedDataFromCsv = parseCsv(rawCsvString, {
      columns: true,
      skip_empty_lines: true//eslint-disable-line camelcase
    });
    let publishedAtByCve = {};
    let cvesWithoutPublishDate = [];
    for (let vuln of parsedDataFromCsv) {
      if (!vuln.published) {// ignore blanks, which can occur  ("published" is empty string)
        cvesWithoutPublishDate.push(vuln.cve);
      }
      // TODO: Confirm timezone of published date and extract JS timestamp accordingly (for now, just using server timezone)
      let publishedAt = vuln.published;
      publishedAtByCve[vuln.cve] = publishedAt;
    }//∞


    const nvdUrl = 'https://services.nvd.nist.gov/rest/json/cves/2.0';
    let allPublishDates = _.keysIn(publishedAtByCve);
    allPublishDates = allPublishDates.sort();
    let startDate = new Date(publishedAtByCve[allPublishDates[allPublishDates.length - 1]]);
    let endDate = new Date(Date.now() + (24 * 60 * 60 * 1000)).toISOString();
    let resultsPerRequest = 1000;
    let index = 0;
    sails.log('Sending API requests to get vulnerabilities published between '+startDate+' and '+endDate+'.');
    await sails.helpers.flow.until(async()=>{
      // Send a request to the NVD API
      let pageOfVulnerabilityData = await sails.helpers.http.get.with({
        url: nvdUrl,
        data: {
          resultsPerPage: resultsPerRequest,
          pubStartDate: startDate,
          pubEndDate: endDate,
          startIndex: index,
        }
      }).intercept((err)=>{
        throw new Error('When sending a request to NVD to get the leatest CVE publish dates, an error occurred. full error:'+err);
      });
      // Add our results to the publishedAtByCve dictionary
      for(let vuln of pageOfVulnerabilityData.vulnerabilities) {
        if(!vuln.cve.published){
          continue;
        }
        // Add the results to the CSV string.
        let publishedAt = vuln.cve.published;
        publishedAtByCve[vuln.cve.id] = publishedAt;
        // rawCsvString += `"${}","${}"\n`;
      }
      let remainingResults = pageOfVulnerabilityData.totalResults - pageOfVulnerabilityData.startIndex;
      // Add the number of results received to the index.
      index += resultsPerRequest;
      // When the amount of remaining results is less than the results per request, we'll stop
      return remainingResults < resultsPerRequest;
    });

    if(cvesWithoutPublishDate.length > 0){
      sails.log('Sending API requests for CVEs that we do not have a publish date for.');
      await sails.helpers.flow.simultaneouslyForEach(cvesWithoutPublishDate, async(vuln)=>{
        let resultsForThisVulnerability = await sails.helpers.http.get.with({
          url: nvdUrl,
          data: {
            cveId: vuln,
          }
        }).retry();

        if(resultsForThisVulnerability.vulnerabilities) {
          let cveDetails = resultsForThisVulnerability.vulnerabilities[0];
          if(cveDetails.cve){
            publishedAtByCve[cveDetails.cve.id] = cveDetails.cve.published;
          }
        }
      });
    }

    let newCsvString = '"cve","published"\n';

    for(let vuln in publishedAtByCve) {
      if(publishedAtByCve[vuln]){
        newCsvString += `"${vuln}","${publishedAtByCve[vuln]}"\n`;
      }
    }

    sails.log('Saving results.');
    // Save the results to the cve-publish-dates file.
    await sails.helpers.fs.write('cve-publish-dates.csv', newCsvString, true);

  }


};

