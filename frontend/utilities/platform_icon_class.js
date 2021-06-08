export const platformIconClass = (platform = "") => {
  if (!platform) return false;

  const lowerPlatform = platform.toLowerCase();

  switch (lowerPlatform) {
    case "macos":
      return "icon-apple-dark-20x20@2x.png";
    case "mac os x":
      return "icon-apple-dark-20x20@2x.png";
    case "mac osx":
      return "icon-apple-dark-20x20@2x.png";
    case "mac os":
      return "icon-apple-dark-20x20@2x.png";
    case "darwin":
      return "icon-apple-dark-20x20@2x.png";
    case "apple":
      return "icon-apple-dark-20x20@2x.png";
    case "centos":
      return "icon-centos-dark-20x20@2x.png";
    case "centos linux":
      return "icon-centos-dark-20x20@2x.png";
    case "ubuntu":
      return "icon-ubuntu-dark-20x20@2x.png";
    case "ubuntu linux":
      return "icon-ubuntu-dark-20x20@2x.png";
    case "linux":
      return "icon-linux-dark-20x20@2x.png";
    case "windows":
      return "icon-windows-dark-20x20@2x.png";
    case "ms windows":
      return "icon-windows-dark-20x20@2x.png";
    default:
      return false;
  }
};

export default platformIconClass;
