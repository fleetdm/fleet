module.exports = {


  friendlyName: 'Cleanup old usage statistics',


  description: 'Deletes HistoricalUsageSnapshot records stored in the database that are over 60 days old.',


  fn: async function () {

    sails.log('Running custom shell script... (`sails run cleanup-old-usage-statistics`)');

    let nowAt = Date.now();
    let sixtyDaysAgoAt = nowAt - (1000 * 60 * 60 * 24 * 60);

    let nativeQueryToDeleteRecords = `
    DELETE FROM "historicalusagesnapshot"
    WHERE "createdAt" < ${sixtyDaysAgoAt}`;

    let queryResult = await sails.sendNativeQuery(nativeQueryToDeleteRecords);
    let numberOfRecordsDeleted = queryResult.rowCount;


    sails.log(`Successfully deleted ${numberOfRecordsDeleted} old HistoricalUsageSnapshot records.`);



  }


};

