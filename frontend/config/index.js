export const environments = {
  development: 'DEV',
  production: 'PROD',
};
const env = process.env.NODE_ENV || environments.development;
const configFileLocation = `./config.${env}.js`;
const settings = require(configFileLocation).default;

export default { environments, settings };
