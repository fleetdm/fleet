interface IValidUrl {
  url: string;
  /**  Validate protocol specified; http validates both http and https */
  protocol?: "http" | "https";
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
