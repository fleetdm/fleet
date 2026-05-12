module.exports = {


  friendlyName: 'View download',


  description: 'Display "Download" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/download'
    }

  },


  fn: async function () {

    let gitHubReponse = await sails.helpers.http.get.with({
      url: 'https://api.github.com/repos/fleetdm/fleet/releases/latest',
      headers: {
        'User-Agent': 'fleet website',
      }
    });

    let downloadAssets = gitHubReponse.assets;


    let macOsDownloadUrl = _.find(downloadAssets, (asset)=>{
      return _.endsWith(asset.browser_download_url, '_macos.zip');
    }).browser_download_url;

    let windowsDownloadUrl = _.find(downloadAssets, (asset)=>{
      return _.endsWith(asset.browser_download_url, '_windows_amd64.zip');
    }).browser_download_url;

    let windowsArmDownloadUrl = _.find(downloadAssets, (asset)=>{
      return _.endsWith(asset.browser_download_url, '_windows_arm64.zip');
    }).browser_download_url;

    return {
      macOsDownloadUrl,
      windowsDownloadUrl,
      windowsArmDownloadUrl,
    };

  }


};
