/**
 * File Upload Settings
 * (sails.config.uploads)
 *
 * These options tell Sails where (and how) to store uploaded files.
 *
 *  > This file is mainly useful for configuring how file uploads in your
 *  > work during development; for example, when lifting on your laptop.
 *  > For recommended production settings, see `config/env/production.js`
 *
 * For all available options, see:
 * https://sailsjs.com/config/uploads
 */

module.exports.uploads = {

  /***************************************************************************
  *                                                                          *
  * Sails apps upload and download to the local disk filesystem by default,  *
  * using a built-in filesystem adapter called `skipper-disk`. This feature  *
  * is mainly intended for convenience during development since, in          *
  * production, many apps will opt to use a different approach for storing   *
  * uploaded files, such as Amazon S3, Azure, or GridFS.                     *
  *                                                                          *
  * Most of the time, the following options should not be changed.           *
  * (Instead, you might want to have a look at `config/env/production.js`.)  *
  *                                                                          *
  ***************************************************************************/
  // bucket: '',// The name of the S3 bucket where software installers will be stored.
  // region: '', // The region where the S3 bucket is located.
  // secret: '', // The secret for the S3 bucket where unassigned software installers will be stored.
  // bucketWithPostfix: '', // This value should be set to the same value as the bucket unless the files are stored in a folder in the S3 bucket. In that case, this value needs to be set to `{bucket name}{folder name}` e.g., unassigned-software-installers/staging
  // prefixForFileDeletion: '', // Only required if the software installers are stored in a folder in the S3 bucket. The name of the folder where the software installers are stored in the S3 bucket with a trailing slash. e.g., staging/

};
