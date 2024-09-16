/**
 * getMimetypeIconClass()
 *
 * -----------------------------------------------------------------
 * @returns {String} the icon-bestowing class name for given MIME type
 */

parasails.registerUtility('getMimetypeIconClass', function getMimetypeIconClass(mimeType) {

  var iconClassName;
  switch(mimeType) {
    case 'application/pdf':
      iconClassName = 'fa fa-file-pdf-o';
      break;
    case 'image/png':
    case 'image/jpeg':
    case 'image/gif':
    case 'application/icon':
    case 'image/svg+xml':
    case 'image/tiff':
      iconClassName = 'fa fa-file-image-o';
      break;
    case 'application/zip':
    case 'application/x-7z-compressed':
    case 'application/x-rar-compressed':
    case 'application/x-tar':
    case 'application/x-bzip':
    case 'application/x-bzip2':
    case 'application/octet-stream':
      iconClassName = 'fa fa-file-archive-o';
      break;
    case 'text/csv':
    case 'application/vnd.ms-excel':
    case 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet':
    case 'application/vnd.oasis.opendocument.spreadsheet':
      iconClassName = 'fa fa-file-excel-o';
      break;
    case 'application/msword':
    case 'application/vnd.openxmlformats-officedocument.wordprocessingml.document':
    case 'application/vnd.oasis.opendocument.text':
    case 'application/rtf':
      iconClassName = 'fa fa-file-word-o';
      break;
    case 'application/vnd.ms-powerpoint':
    case 'application/vnd.openxmlformats-officedocument.presentationml.presentation':
    case 'application/vnd.oasis.opendocument.presentation':
      iconClassName = 'fa fa-file-powerpoint-o';
      break;
    case 'application/vnd.ms-outlook':
      iconClassName = 'fa fa-envelope-o';
      break;
    case 'text/plain':
      iconClassName = 'fa fa-file-text-o';
      break;
    default:
      if(mimeType && mimeType.match(/^image/)) {
        iconClassName = 'fa fa-file-image-o';
      } else if(mimeType && mimeType.match(/^video/)) {
        iconClassName = 'fa fa-file-video-o';
      } else if(mimeType && mimeType.match(/^audio/)) {
        iconClassName = 'fa fa-file-audio-o';
      } else if(mimeType && mimeType.match(/^text/)) {
        iconClassName = 'fa fa-file-text-o';
      } else {
        iconClassName = 'fa fa-file-o';
      }
      break;
  }

  return iconClassName;

});
