import { redirect } from "next/navigation";

export const instant = false;

export default function ArchitectureIndex() {
  redirect("/architecture/maps");
}
