import config from '../config';

const { window } = global;
const { settings } = config;

export default {
  getItem: (itemName) => {
    const { localStorage } = window;
    const { env } = settings;

    return localStorage.getItem(`KOLIDE-${env}::${itemName}`);
  },
  setItem: (itemName, value) => {
    const { localStorage } = window;
    const { env } = settings;

    return localStorage.setItem(`KOLIDE-${env}::${itemName}`, value);
  },
};
