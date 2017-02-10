import moment from 'moment';

const shouldNagUser = ({ license }) => {
  const { allowed_hosts: allowedHosts, expiry, hosts, revoked } = license;

  const hostsOverenrolled = hosts > allowedHosts;
  const licenseExpired = moment().isAfter(moment(expiry));

  return hostsOverenrolled || licenseExpired || revoked;
};

export default { shouldNagUser };
