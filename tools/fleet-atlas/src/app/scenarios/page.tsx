import ScenariosClient from "./ScenariosClient";
import { listCustomScenarios } from "@/lib/custom-scenarios";

export const instant = false;

export default function ScenariosIndex() {
  const custom = listCustomScenarios();
  return <ScenariosClient customScenarios={custom} />;
}
