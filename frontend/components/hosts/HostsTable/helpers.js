import moment from 'moment';

const BYTES_PER_GIGABYTE = 1074000000;
const NANOSECONDS_PER_MILLISECOND = 1000000;

const inGigaBytes = (bytes) => {
  return (bytes / BYTES_PER_GIGABYTE).toFixed(1);
};

const inMilliseconds = (nanoseconds) => {
  return nanoseconds / NANOSECONDS_PER_MILLISECOND;
};

export const humanUptime = (uptimeInNanoseconds) => {
  const milliseconds = inMilliseconds(uptimeInNanoseconds);

  return moment.duration(milliseconds, 'milliseconds').humanize();
};

export const humanLastSeen = (lastSeen) => {
  return moment(lastSeen).format('YYYY-MM-DD HH:mm:ss');
};

export const humanMemory = (bytes) => {
  return `${inGigaBytes(bytes)} GB`;
};

export default { humanMemory, humanUptime };
