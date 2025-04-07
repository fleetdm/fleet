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
      });
      let filename = packageInformation.software_title.software_package.name;
      // [?]: https://fleetdm.com/docs/rest-api/rest-api#download-package
      // GET /api/v1/fleet/software/titles/:software_title_id/package?team_id=${teamId}
      downloading = await sails.helpers.http.getStream.with({
        url: `${sails.config.custom.fleetBaseUrl}/api/v1/fleet/software/titles/${fleetApid}/package?alt=media&team_id=${teamApid}`,
        headers: {
          Authorization: `Bearer ${sails.config.custom.fleetApiToken}`,
        }
      });
      this.res.attachment(filename);
    }
    return downloading;
  }


};
