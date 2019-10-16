// Encapsulate the URL Prefix so that this is the only module that
// needs to access the global. All other modules should use this one.

export default window.urlPrefix || '';
