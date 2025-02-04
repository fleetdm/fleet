module.exports = {


  friendlyName: 'Download software',


  description: 'Downloads a software installer for deployed or undeployed software.',


  inputs: {
    id: {
      type: 'number',
      description: 'The database ID of the undeployed software to download.'
    },
    fleetApid: {
      type: 'string',
      description: 'The fleetApid of a software on a team.'
    },
    teamApid: {
      type: 'string',
      description: 'The team API ID of a team that the software is deployed to.'
    }
  },


  exits: {
    success: {
      outputFriendlyName: 'File',
      outputDescription: 'The streaming bytes of the file.',
      outputType: 'ref'
    },

    notFound: {
      description: 'No software exists with the specified ID.',
      responseType: 'notFound'
    },
  },


  fn: async function ({id, fleetApid, teamApid}) {
    if(!fleetApid && !id){
      return this.res.badRequest();
    }
    let downloading;

    if(id){
      let softwareToDownload = await UndeployedSoftware.findOne({id: id});
      if(!softwareToDownload){
        throw 'notFound';
      }
      downloading = await sails.startDownload(softwareToDownload.uploadFd, {bucket: sails.config.uploads.bucketWithPostfix});
      this.res.type(softwareToDownload.uploadMime);
      this.res.attachment(softwareToDownload.name);
    } else {
      // Get information about the installer package from the Fleet server.
      let packageInformation = await sails.helpers.http.get.with({
        url: `${sails.config.custom.fleetBaseUrl}/api/latest/fleet/software/titles/${fleetApid}?team_id=${teamApid}&available_for_install=true`,
        headers: {
          Authorization: `Bearer ${sails.config.custom.fleetApiToken}`,
        }
      }).intercept({raw: {statusCode: 404}}, (error)=>{
        // If the Fleet instance returns a 404 response, log a warning and display the notFound response page.
        sails.log.warn(`When attempting to get information about a software title (id: ${fleetApid}) for team_id ${teamApid}, the Fleet instance returned a 404 response. Full Error: ${require('util').inspect(error, {depth: 1})}`);
        return 'notFound';
      });
      let filename = packageInformation.software_title.software_package.name;
      // [?]: https://fleetdm.com/docs/rest-api/rest-api#download-package
      // GET /api/v1/fleet/software/titles/:software_title_id/package?team_id=${teamId}
      downloading = await sails.helpers.http.getStream.with({
        url: `${sails.config.custom.fleetBaseUrl}/api/v1/fleet/software/titles/${fleetApid}/package?alt=media&team_id=${teamApid}`,
        headers: {
          Authorization: `Bearer ${sails.config.custom.fleetApiToken}`,
        }
      })
      .intercept({raw: {statusCode: 404}}, (error)=>{
        // If the installer is missing, throw an error with information about this software installer/title.
        return new Error(`When attempting to download the installer for ${filename} (id: ${fleetApid}), the Fleet instance returned a 404 response when a request was sent to get a download stream of the installer on team_id ${teamApid}. Full Error: ${require('util').inspect(error, {depth: 1})}`);
      });
      this.res.attachment(filename);
    }
    return downloading;
  }


};
