const { window } = global;

const local = {
  clear: () => {
    const { localStorage } = window;

    localStorage.clear();
  },
  getItem: (itemName: string) => {
    const { localStorage } = window;

    return localStorage.getItem(`FLEET::${itemName}`);
  },
  setItem: (itemName: string, value: any) => {
    const { localStorage } = window;

    return localStorage.setItem(`FLEET::${itemName}`, value);
  },
  removeItem: (itemName: string) => {
    const { localStorage } = window;

    localStorage.removeItem(`FLEET::${itemName}`);
  },
};

export const authToken = () => {
  return local.getItem("auth_token");
};

export default local;
