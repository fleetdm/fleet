import { redirect } from "next/navigation";

export const instant = false;

export default function BackgroundIndex() {
  redirect("/background/cron");
}
