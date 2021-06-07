const { window } = global;

const local = {
  clear: () => {
    const { localStorage } = window;

    localStorage.clear();
  },
  getItem: (itemName) => {
    const { localStorage } = window;

    return localStorage.getItem(`FLEET::${itemName}`);
  },
  setItem: (itemName, value) => {
    const { localStorage } = window;

    return localStorage.setItem(`FLEET::${itemName}`, value);
  },
};

export const authToken = () => {
  return local.getItem("auth_token");
};

export default local;
