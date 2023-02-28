// eslint-disable-next-line import/prefer-default-export
const getHostStatusTooltipText = (status: string): string => {
  if (status === "online") {
    return "Online hosts will respond to a live query.";
  }
  return "Offline hosts won’t respond to a live query because they may be shut down, asleep, or not connected to the internet.";
};

export default getHostStatusTooltipText;
