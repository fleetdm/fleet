import moment from 'moment';

const BYTES_PER_GIGABYTE = 1000000000;
const NANOSECONDS_PER_MILLISECOND = 1000000;

const inGigaBytes = (bytes) => {
  return (bytes / BYTES_PER_GIGABYTE).toFixed(2);
};

const inMilliseconds = (nanoseconds) => {
  return nanoseconds / NANOSECONDS_PER_MILLISECOND;
};

export const humanUptime = (uptimeInNanoseconds) => {
  const milliseconds = inMilliseconds(uptimeInNanoseconds);

  return moment.duration(milliseconds, 'milliseconds').humanize();
};

export const humanMemory = (bytes) => {
  return `${inGigaBytes(bytes)} GB`;
};

export const platformIconClass = (platform) => {
  let platformClass = platform.toLowerCase();

  if (platformClass === 'darwin') {
    platformClass = 'apple';
  }

  return `kolidecon-${platformClass}`;
};

export default { humanUptime, platformIconClass };
