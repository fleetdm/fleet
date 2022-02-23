const { window } = global;

const local = {
  clear: (): void => {
    const { localStorage } = window;

    localStorage.clear();
  },
  getItem: (itemName: string): string | null => {
    const { localStorage } = window;

    return localStorage.getItem(`FLEET::${itemName}`);
  },
  setItem: (itemName: string, value: string): void => {
    const { localStorage } = window;

    return localStorage.setItem(`FLEET::${itemName}`, value);
  },
  removeItem: (itemName: string): void => {
    const { localStorage } = window;

    localStorage.removeItem(`FLEET::${itemName}`);
  },
};

export const authToken = (): string | null => {
  return local.getItem("auth_token");
};

export const clearToken = (): void => {
  return local.removeItem("auth_token");
};

export default local;
