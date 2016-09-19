import config from '../config';

const { window } = global;
const { settings } = config;

const local = {
  clear: () => {
    const { localStorage } = window;

    localStorage.clear();
  },
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

export const authToken = () => { return local.getItem('auth_token'); };

export default local;
