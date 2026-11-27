import { fleetFileExists } from "@/lib/fleet-repo";
import MapsClient from "./MapsClient";

export const instant = false;

export default async function MapsPage() {
  const connected = await fleetFileExists("go.mod");
  return <MapsClient connected={connected} />;
}
