interface IValidUrl {
  url: string;
  isHttp?: boolean;
}

export default ({ url, isHttp = false }: IValidUrl): boolean => {
  try {
    const newUrl = new URL(url);
    if (isHttp) {
      return newUrl.protocol === "http:" || newUrl.protocol === "https:";
    }
    return true;
  } catch (e) {
    return false;
  }
};
