import { redirect } from "next/navigation";

export const instant = false;

export default function GlossaryIndex() {
  redirect("/glossary/terms");
}
