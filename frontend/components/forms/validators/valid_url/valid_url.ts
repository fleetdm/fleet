interface IValidUrl {
  url: string;
  protocol?: string;
}

export default ({ url, protocol }: IValidUrl): boolean => {
  try {
    const newUrl = new URL(url);
    if (protocol === "http") {
      return newUrl.protocol === "http:" || newUrl.protocol === "https:";
    }
    if (protocol === "https") {
      return newUrl.protocol === "https:";
    }
    return true;
  } catch (e) {
    return false;
  }
};
