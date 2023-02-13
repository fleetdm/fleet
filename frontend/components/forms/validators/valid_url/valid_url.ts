export default (url: string, isHttp = "false"): boolean => {
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
