export default function getHostStatusTooltipText(status: string): string {
  switch (status) {
    case "online":
      return "Online hosts will respond to a live query.";
    case "offline":
      return "Offline hosts won’t respond to a live query because they may be shut down, asleep, or not connected to the internet.";
    default:
      return "";
  }
}
