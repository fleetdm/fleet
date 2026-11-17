import { redirect } from "next/navigation";

export const instant = false;

export default function Home() {
  redirect("/architecture/maps");
}
