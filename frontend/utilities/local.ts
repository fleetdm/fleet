const {
  window: { localStorage },
} = global;

const local = {
  clear: (): void => {
    localStorage.clear();
  },
  getItem: (itemName: string): string | null => {
    return localStorage.getItem(`FLEET::${itemName}`);
  },
  setItem: (itemName: string, value: string): void => {
    return localStorage.setItem(`FLEET::${itemName}`, value);
  },
  removeItem: (itemName: string): void => {
    localStorage.removeItem(`FLEET::${itemName}`);
  },
};

export default local;
